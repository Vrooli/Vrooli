export const popular_findMany = {
  "fieldName": "populars",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "populars",
        "loc": {
          "start": 7840,
          "end": 7848
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 7849,
              "end": 7854
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 7857,
                "end": 7862
              }
            },
            "loc": {
              "start": 7856,
              "end": 7862
            }
          },
          "loc": {
            "start": 7849,
            "end": 7862
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "edges",
              "loc": {
                "start": 7870,
                "end": 7875
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "cursor",
                    "loc": {
                      "start": 7886,
                      "end": 7892
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7886,
                    "end": 7892
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 7901,
                      "end": 7905
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Api",
                            "loc": {
                              "start": 7927,
                              "end": 7930
                            }
                          },
                          "loc": {
                            "start": 7927,
                            "end": 7930
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Api_list",
                                "loc": {
                                  "start": 7952,
                                  "end": 7960
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 7949,
                                "end": 7960
                              }
                            }
                          ],
                          "loc": {
                            "start": 7931,
                            "end": 7974
                          }
                        },
                        "loc": {
                          "start": 7920,
                          "end": 7974
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Code",
                            "loc": {
                              "start": 7994,
                              "end": 7998
                            }
                          },
                          "loc": {
                            "start": 7994,
                            "end": 7998
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Code_list",
                                "loc": {
                                  "start": 8020,
                                  "end": 8029
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8017,
                                "end": 8029
                              }
                            }
                          ],
                          "loc": {
                            "start": 7999,
                            "end": 8043
                          }
                        },
                        "loc": {
                          "start": 7987,
                          "end": 8043
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Note",
                            "loc": {
                              "start": 8063,
                              "end": 8067
                            }
                          },
                          "loc": {
                            "start": 8063,
                            "end": 8067
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Note_list",
                                "loc": {
                                  "start": 8089,
                                  "end": 8098
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8086,
                                "end": 8098
                              }
                            }
                          ],
                          "loc": {
                            "start": 8068,
                            "end": 8112
                          }
                        },
                        "loc": {
                          "start": 8056,
                          "end": 8112
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Project",
                            "loc": {
                              "start": 8132,
                              "end": 8139
                            }
                          },
                          "loc": {
                            "start": 8132,
                            "end": 8139
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Project_list",
                                "loc": {
                                  "start": 8161,
                                  "end": 8173
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8158,
                                "end": 8173
                              }
                            }
                          ],
                          "loc": {
                            "start": 8140,
                            "end": 8187
                          }
                        },
                        "loc": {
                          "start": 8125,
                          "end": 8187
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Question",
                            "loc": {
                              "start": 8207,
                              "end": 8215
                            }
                          },
                          "loc": {
                            "start": 8207,
                            "end": 8215
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Question_list",
                                "loc": {
                                  "start": 8237,
                                  "end": 8250
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8234,
                                "end": 8250
                              }
                            }
                          ],
                          "loc": {
                            "start": 8216,
                            "end": 8264
                          }
                        },
                        "loc": {
                          "start": 8200,
                          "end": 8264
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Routine",
                            "loc": {
                              "start": 8284,
                              "end": 8291
                            }
                          },
                          "loc": {
                            "start": 8284,
                            "end": 8291
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Routine_list",
                                "loc": {
                                  "start": 8313,
                                  "end": 8325
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8310,
                                "end": 8325
                              }
                            }
                          ],
                          "loc": {
                            "start": 8292,
                            "end": 8339
                          }
                        },
                        "loc": {
                          "start": 8277,
                          "end": 8339
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Standard",
                            "loc": {
                              "start": 8359,
                              "end": 8367
                            }
                          },
                          "loc": {
                            "start": 8359,
                            "end": 8367
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Standard_list",
                                "loc": {
                                  "start": 8389,
                                  "end": 8402
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8386,
                                "end": 8402
                              }
                            }
                          ],
                          "loc": {
                            "start": 8368,
                            "end": 8416
                          }
                        },
                        "loc": {
                          "start": 8352,
                          "end": 8416
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Team",
                            "loc": {
                              "start": 8436,
                              "end": 8440
                            }
                          },
                          "loc": {
                            "start": 8436,
                            "end": 8440
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Team_list",
                                "loc": {
                                  "start": 8462,
                                  "end": 8471
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8459,
                                "end": 8471
                              }
                            }
                          ],
                          "loc": {
                            "start": 8441,
                            "end": 8485
                          }
                        },
                        "loc": {
                          "start": 8429,
                          "end": 8485
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "User",
                            "loc": {
                              "start": 8505,
                              "end": 8509
                            }
                          },
                          "loc": {
                            "start": 8505,
                            "end": 8509
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "User_list",
                                "loc": {
                                  "start": 8531,
                                  "end": 8540
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8528,
                                "end": 8540
                              }
                            }
                          ],
                          "loc": {
                            "start": 8510,
                            "end": 8554
                          }
                        },
                        "loc": {
                          "start": 8498,
                          "end": 8554
                        }
                      }
                    ],
                    "loc": {
                      "start": 7906,
                      "end": 8564
                    }
                  },
                  "loc": {
                    "start": 7901,
                    "end": 8564
                  }
                }
              ],
              "loc": {
                "start": 7876,
                "end": 8570
              }
            },
            "loc": {
              "start": 7870,
              "end": 8570
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 8575,
                "end": 8583
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasNextPage",
                    "loc": {
                      "start": 8594,
                      "end": 8605
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8594,
                    "end": 8605
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorApi",
                    "loc": {
                      "start": 8614,
                      "end": 8626
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8614,
                    "end": 8626
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorCode",
                    "loc": {
                      "start": 8635,
                      "end": 8648
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8635,
                    "end": 8648
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorNote",
                    "loc": {
                      "start": 8657,
                      "end": 8670
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8657,
                    "end": 8670
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 8679,
                      "end": 8695
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8679,
                    "end": 8695
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorQuestion",
                    "loc": {
                      "start": 8704,
                      "end": 8721
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8704,
                    "end": 8721
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRoutine",
                    "loc": {
                      "start": 8730,
                      "end": 8746
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8730,
                    "end": 8746
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorStandard",
                    "loc": {
                      "start": 8755,
                      "end": 8772
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8755,
                    "end": 8772
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorTeam",
                    "loc": {
                      "start": 8781,
                      "end": 8794
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8781,
                    "end": 8794
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorUser",
                    "loc": {
                      "start": 8803,
                      "end": 8816
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8803,
                    "end": 8816
                  }
                }
              ],
              "loc": {
                "start": 8584,
                "end": 8822
              }
            },
            "loc": {
              "start": 8575,
              "end": 8822
            }
          }
        ],
        "loc": {
          "start": 7864,
          "end": 8826
        }
      },
      "loc": {
        "start": 7840,
        "end": 8826
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 28,
          "end": 36
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 43,
                "end": 55
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 66,
                      "end": 68
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 66,
                    "end": 68
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 77,
                      "end": 85
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 77,
                    "end": 85
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "details",
                    "loc": {
                      "start": 94,
                      "end": 101
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 94,
                    "end": 101
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 110,
                      "end": 114
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 110,
                    "end": 114
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "summary",
                    "loc": {
                      "start": 123,
                      "end": 130
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 123,
                    "end": 130
                  }
                }
              ],
              "loc": {
                "start": 56,
                "end": 136
              }
            },
            "loc": {
              "start": 43,
              "end": 136
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 141,
                "end": 143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 141,
              "end": 143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 148,
                "end": 158
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 148,
              "end": 158
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 163,
                "end": 173
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 163,
              "end": 173
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "callLink",
              "loc": {
                "start": 178,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 178,
              "end": 186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 191,
                "end": 204
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 191,
              "end": 204
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "documentationLink",
              "loc": {
                "start": 209,
                "end": 226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 209,
              "end": 226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 231,
                "end": 241
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 231,
              "end": 241
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 246,
                "end": 254
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 246,
              "end": 254
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 259,
                "end": 268
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 259,
              "end": 268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 273,
                "end": 285
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 273,
              "end": 285
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 290,
                "end": 302
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 290,
              "end": 302
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 307,
                "end": 319
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 307,
              "end": 319
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 324,
                "end": 327
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 338,
                      "end": 348
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 338,
                    "end": 348
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 357,
                      "end": 364
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 357,
                    "end": 364
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 373,
                      "end": 382
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 373,
                    "end": 382
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 391,
                      "end": 400
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 391,
                    "end": 400
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 409,
                      "end": 418
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 409,
                    "end": 418
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 427,
                      "end": 433
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 427,
                    "end": 433
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 442,
                      "end": 449
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 442,
                    "end": 449
                  }
                }
              ],
              "loc": {
                "start": 328,
                "end": 455
              }
            },
            "loc": {
              "start": 324,
              "end": 455
            }
          }
        ],
        "loc": {
          "start": 37,
          "end": 457
        }
      },
      "loc": {
        "start": 28,
        "end": 457
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 458,
          "end": 460
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 458,
        "end": 460
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 461,
          "end": 471
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 461,
        "end": 471
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 472,
          "end": 482
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 472,
        "end": 482
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 483,
          "end": 492
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 483,
        "end": 492
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 493,
          "end": 504
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 493,
        "end": 504
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 505,
          "end": 511
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 521,
                "end": 531
              }
            },
            "directives": [],
            "loc": {
              "start": 518,
              "end": 531
            }
          }
        ],
        "loc": {
          "start": 512,
          "end": 533
        }
      },
      "loc": {
        "start": 505,
        "end": 533
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 534,
          "end": 539
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 553,
                  "end": 557
                }
              },
              "loc": {
                "start": 553,
                "end": 557
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 571,
                      "end": 579
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 568,
                    "end": 579
                  }
                }
              ],
              "loc": {
                "start": 558,
                "end": 585
              }
            },
            "loc": {
              "start": 546,
              "end": 585
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 597,
                  "end": 601
                }
              },
              "loc": {
                "start": 597,
                "end": 601
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 615,
                      "end": 623
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 612,
                    "end": 623
                  }
                }
              ],
              "loc": {
                "start": 602,
                "end": 629
              }
            },
            "loc": {
              "start": 590,
              "end": 629
            }
          }
        ],
        "loc": {
          "start": 540,
          "end": 631
        }
      },
      "loc": {
        "start": 534,
        "end": 631
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 632,
          "end": 643
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 632,
        "end": 643
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 644,
          "end": 658
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 644,
        "end": 658
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 659,
          "end": 664
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 659,
        "end": 664
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 665,
          "end": 674
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 665,
        "end": 674
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 675,
          "end": 679
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 689,
                "end": 697
              }
            },
            "directives": [],
            "loc": {
              "start": 686,
              "end": 697
            }
          }
        ],
        "loc": {
          "start": 680,
          "end": 699
        }
      },
      "loc": {
        "start": 675,
        "end": 699
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 700,
          "end": 714
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 700,
        "end": 714
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 715,
          "end": 720
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 715,
        "end": 720
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 721,
          "end": 724
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 731,
                "end": 740
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 731,
              "end": 740
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 745,
                "end": 756
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 745,
              "end": 756
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 761,
                "end": 772
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 761,
              "end": 772
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 777,
                "end": 786
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 777,
              "end": 786
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 791,
                "end": 798
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 791,
              "end": 798
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 803,
                "end": 811
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 803,
              "end": 811
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 816,
                "end": 828
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 816,
              "end": 828
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 833,
                "end": 841
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 833,
              "end": 841
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 846,
                "end": 854
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 846,
              "end": 854
            }
          }
        ],
        "loc": {
          "start": 725,
          "end": 856
        }
      },
      "loc": {
        "start": 721,
        "end": 856
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 885,
          "end": 887
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 885,
        "end": 887
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 888,
          "end": 897
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 888,
        "end": 897
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 929,
          "end": 937
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 944,
                "end": 956
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 967,
                      "end": 969
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 967,
                    "end": 969
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 978,
                      "end": 986
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 978,
                    "end": 986
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 995,
                      "end": 1006
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 995,
                    "end": 1006
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 1015,
                      "end": 1027
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1015,
                    "end": 1027
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1036,
                      "end": 1040
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1036,
                    "end": 1040
                  }
                }
              ],
              "loc": {
                "start": 957,
                "end": 1046
              }
            },
            "loc": {
              "start": 944,
              "end": 1046
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1051,
                "end": 1053
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1051,
              "end": 1053
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1058,
                "end": 1068
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1058,
              "end": 1068
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1073,
                "end": 1083
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1073,
              "end": 1083
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1088,
                "end": 1098
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1088,
              "end": 1098
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 1103,
                "end": 1112
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1103,
              "end": 1112
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1117,
                "end": 1125
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1117,
              "end": 1125
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1130,
                "end": 1139
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1130,
              "end": 1139
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "codeLanguage",
              "loc": {
                "start": 1144,
                "end": 1156
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1144,
              "end": 1156
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "codeType",
              "loc": {
                "start": 1161,
                "end": 1169
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1161,
              "end": 1169
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 1174,
                "end": 1181
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1174,
              "end": 1181
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1186,
                "end": 1198
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1186,
              "end": 1198
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1203,
                "end": 1215
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1203,
              "end": 1215
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 1220,
                "end": 1233
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1220,
              "end": 1233
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 1238,
                "end": 1260
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1238,
              "end": 1260
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 1265,
                "end": 1275
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1265,
              "end": 1275
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1280,
                "end": 1292
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1280,
              "end": 1292
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1297,
                "end": 1300
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 1311,
                      "end": 1321
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1311,
                    "end": 1321
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 1330,
                      "end": 1337
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1330,
                    "end": 1337
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1346,
                      "end": 1355
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1346,
                    "end": 1355
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1364,
                      "end": 1373
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1364,
                    "end": 1373
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1382,
                      "end": 1391
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1382,
                    "end": 1391
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 1400,
                      "end": 1406
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1400,
                    "end": 1406
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1415,
                      "end": 1422
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1415,
                    "end": 1422
                  }
                }
              ],
              "loc": {
                "start": 1301,
                "end": 1428
              }
            },
            "loc": {
              "start": 1297,
              "end": 1428
            }
          }
        ],
        "loc": {
          "start": 938,
          "end": 1430
        }
      },
      "loc": {
        "start": 929,
        "end": 1430
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1431,
          "end": 1433
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1431,
        "end": 1433
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1434,
          "end": 1444
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1434,
        "end": 1444
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1445,
          "end": 1455
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1445,
        "end": 1455
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1456,
          "end": 1465
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1456,
        "end": 1465
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 1466,
          "end": 1477
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1466,
        "end": 1477
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1478,
          "end": 1484
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 1494,
                "end": 1504
              }
            },
            "directives": [],
            "loc": {
              "start": 1491,
              "end": 1504
            }
          }
        ],
        "loc": {
          "start": 1485,
          "end": 1506
        }
      },
      "loc": {
        "start": 1478,
        "end": 1506
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1507,
          "end": 1512
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 1526,
                  "end": 1530
                }
              },
              "loc": {
                "start": 1526,
                "end": 1530
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 1544,
                      "end": 1552
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1541,
                    "end": 1552
                  }
                }
              ],
              "loc": {
                "start": 1531,
                "end": 1558
              }
            },
            "loc": {
              "start": 1519,
              "end": 1558
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 1570,
                  "end": 1574
                }
              },
              "loc": {
                "start": 1570,
                "end": 1574
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 1588,
                      "end": 1596
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1585,
                    "end": 1596
                  }
                }
              ],
              "loc": {
                "start": 1575,
                "end": 1602
              }
            },
            "loc": {
              "start": 1563,
              "end": 1602
            }
          }
        ],
        "loc": {
          "start": 1513,
          "end": 1604
        }
      },
      "loc": {
        "start": 1507,
        "end": 1604
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1605,
          "end": 1616
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1605,
        "end": 1616
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 1617,
          "end": 1631
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1617,
        "end": 1631
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 1632,
          "end": 1637
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1632,
        "end": 1637
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1638,
          "end": 1647
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1638,
        "end": 1647
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1648,
          "end": 1652
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 1662,
                "end": 1670
              }
            },
            "directives": [],
            "loc": {
              "start": 1659,
              "end": 1670
            }
          }
        ],
        "loc": {
          "start": 1653,
          "end": 1672
        }
      },
      "loc": {
        "start": 1648,
        "end": 1672
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 1673,
          "end": 1687
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1673,
        "end": 1687
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 1688,
          "end": 1693
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1688,
        "end": 1693
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1694,
          "end": 1697
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1704,
                "end": 1713
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1704,
              "end": 1713
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1718,
                "end": 1729
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1718,
              "end": 1729
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 1734,
                "end": 1745
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1734,
              "end": 1745
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1750,
                "end": 1759
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1750,
              "end": 1759
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1764,
                "end": 1771
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1764,
              "end": 1771
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1776,
                "end": 1784
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1776,
              "end": 1784
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1789,
                "end": 1801
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1789,
              "end": 1801
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1806,
                "end": 1814
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1806,
              "end": 1814
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1819,
                "end": 1827
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1819,
              "end": 1827
            }
          }
        ],
        "loc": {
          "start": 1698,
          "end": 1829
        }
      },
      "loc": {
        "start": 1694,
        "end": 1829
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1860,
          "end": 1862
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1860,
        "end": 1862
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1863,
          "end": 1872
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1863,
        "end": 1872
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1906,
          "end": 1908
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1906,
        "end": 1908
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1909,
          "end": 1919
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1909,
        "end": 1919
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1920,
          "end": 1930
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1920,
        "end": 1930
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 1931,
          "end": 1936
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1931,
        "end": 1936
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 1937,
          "end": 1942
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1937,
        "end": 1942
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1943,
          "end": 1948
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 1962,
                  "end": 1966
                }
              },
              "loc": {
                "start": 1962,
                "end": 1966
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 1980,
                      "end": 1988
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1977,
                    "end": 1988
                  }
                }
              ],
              "loc": {
                "start": 1967,
                "end": 1994
              }
            },
            "loc": {
              "start": 1955,
              "end": 1994
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 2006,
                  "end": 2010
                }
              },
              "loc": {
                "start": 2006,
                "end": 2010
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 2024,
                      "end": 2032
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2021,
                    "end": 2032
                  }
                }
              ],
              "loc": {
                "start": 2011,
                "end": 2038
              }
            },
            "loc": {
              "start": 1999,
              "end": 2038
            }
          }
        ],
        "loc": {
          "start": 1949,
          "end": 2040
        }
      },
      "loc": {
        "start": 1943,
        "end": 2040
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2041,
          "end": 2044
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2051,
                "end": 2060
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2051,
              "end": 2060
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2065,
                "end": 2074
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2065,
              "end": 2074
            }
          }
        ],
        "loc": {
          "start": 2045,
          "end": 2076
        }
      },
      "loc": {
        "start": 2041,
        "end": 2076
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 2108,
          "end": 2116
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2123,
                "end": 2135
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2146,
                      "end": 2148
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2146,
                    "end": 2148
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2157,
                      "end": 2165
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2157,
                    "end": 2165
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2174,
                      "end": 2185
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2174,
                    "end": 2185
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2194,
                      "end": 2198
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2194,
                    "end": 2198
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "pages",
                    "loc": {
                      "start": 2207,
                      "end": 2212
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2227,
                            "end": 2229
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2227,
                          "end": 2229
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pageIndex",
                          "loc": {
                            "start": 2242,
                            "end": 2251
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2242,
                          "end": 2251
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 2264,
                            "end": 2268
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2264,
                          "end": 2268
                        }
                      }
                    ],
                    "loc": {
                      "start": 2213,
                      "end": 2278
                    }
                  },
                  "loc": {
                    "start": 2207,
                    "end": 2278
                  }
                }
              ],
              "loc": {
                "start": 2136,
                "end": 2284
              }
            },
            "loc": {
              "start": 2123,
              "end": 2284
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2289,
                "end": 2291
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2289,
              "end": 2291
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2296,
                "end": 2306
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2296,
              "end": 2306
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2311,
                "end": 2321
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2311,
              "end": 2321
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 2326,
                "end": 2334
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2326,
              "end": 2334
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2339,
                "end": 2348
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2339,
              "end": 2348
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 2353,
                "end": 2365
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2353,
              "end": 2365
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 2370,
                "end": 2382
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2370,
              "end": 2382
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 2387,
                "end": 2399
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2387,
              "end": 2399
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2404,
                "end": 2407
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 2418,
                      "end": 2428
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2418,
                    "end": 2428
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 2437,
                      "end": 2444
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2437,
                    "end": 2444
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2453,
                      "end": 2462
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2453,
                    "end": 2462
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2471,
                      "end": 2480
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2471,
                    "end": 2480
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2489,
                      "end": 2498
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2489,
                    "end": 2498
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 2507,
                      "end": 2513
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2507,
                    "end": 2513
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2522,
                      "end": 2529
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2522,
                    "end": 2529
                  }
                }
              ],
              "loc": {
                "start": 2408,
                "end": 2535
              }
            },
            "loc": {
              "start": 2404,
              "end": 2535
            }
          }
        ],
        "loc": {
          "start": 2117,
          "end": 2537
        }
      },
      "loc": {
        "start": 2108,
        "end": 2537
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2538,
          "end": 2540
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2538,
        "end": 2540
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2541,
          "end": 2551
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2541,
        "end": 2551
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2552,
          "end": 2562
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2552,
        "end": 2562
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2563,
          "end": 2572
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2563,
        "end": 2572
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 2573,
          "end": 2584
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2573,
        "end": 2584
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 2585,
          "end": 2591
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 2601,
                "end": 2611
              }
            },
            "directives": [],
            "loc": {
              "start": 2598,
              "end": 2611
            }
          }
        ],
        "loc": {
          "start": 2592,
          "end": 2613
        }
      },
      "loc": {
        "start": 2585,
        "end": 2613
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 2614,
          "end": 2619
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 2633,
                  "end": 2637
                }
              },
              "loc": {
                "start": 2633,
                "end": 2637
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 2651,
                      "end": 2659
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2648,
                    "end": 2659
                  }
                }
              ],
              "loc": {
                "start": 2638,
                "end": 2665
              }
            },
            "loc": {
              "start": 2626,
              "end": 2665
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 2677,
                  "end": 2681
                }
              },
              "loc": {
                "start": 2677,
                "end": 2681
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 2695,
                      "end": 2703
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2692,
                    "end": 2703
                  }
                }
              ],
              "loc": {
                "start": 2682,
                "end": 2709
              }
            },
            "loc": {
              "start": 2670,
              "end": 2709
            }
          }
        ],
        "loc": {
          "start": 2620,
          "end": 2711
        }
      },
      "loc": {
        "start": 2614,
        "end": 2711
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 2712,
          "end": 2723
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2712,
        "end": 2723
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 2724,
          "end": 2738
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2724,
        "end": 2738
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 2739,
          "end": 2744
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2739,
        "end": 2744
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2745,
          "end": 2754
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2745,
        "end": 2754
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 2755,
          "end": 2759
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 2769,
                "end": 2777
              }
            },
            "directives": [],
            "loc": {
              "start": 2766,
              "end": 2777
            }
          }
        ],
        "loc": {
          "start": 2760,
          "end": 2779
        }
      },
      "loc": {
        "start": 2755,
        "end": 2779
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 2780,
          "end": 2794
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2780,
        "end": 2794
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 2795,
          "end": 2800
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2795,
        "end": 2800
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2801,
          "end": 2804
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2811,
                "end": 2820
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2811,
              "end": 2820
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2825,
                "end": 2836
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2825,
              "end": 2836
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 2841,
                "end": 2852
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2841,
              "end": 2852
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2857,
                "end": 2866
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2857,
              "end": 2866
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2871,
                "end": 2878
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2871,
              "end": 2878
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 2883,
                "end": 2891
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2883,
              "end": 2891
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2896,
                "end": 2908
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2896,
              "end": 2908
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2913,
                "end": 2921
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2913,
              "end": 2921
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 2926,
                "end": 2934
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2926,
              "end": 2934
            }
          }
        ],
        "loc": {
          "start": 2805,
          "end": 2936
        }
      },
      "loc": {
        "start": 2801,
        "end": 2936
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2967,
          "end": 2969
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2967,
        "end": 2969
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2970,
          "end": 2979
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2970,
        "end": 2979
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 3017,
          "end": 3025
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 3032,
                "end": 3044
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3055,
                      "end": 3057
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3055,
                    "end": 3057
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3066,
                      "end": 3074
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3066,
                    "end": 3074
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3083,
                      "end": 3094
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3083,
                    "end": 3094
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3103,
                      "end": 3107
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3103,
                    "end": 3107
                  }
                }
              ],
              "loc": {
                "start": 3045,
                "end": 3113
              }
            },
            "loc": {
              "start": 3032,
              "end": 3113
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3118,
                "end": 3120
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3118,
              "end": 3120
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3125,
                "end": 3135
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3125,
              "end": 3135
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3140,
                "end": 3150
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3140,
              "end": 3150
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 3155,
                "end": 3171
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3155,
              "end": 3171
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 3176,
                "end": 3184
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3176,
              "end": 3184
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3189,
                "end": 3198
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3189,
              "end": 3198
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 3203,
                "end": 3215
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3203,
              "end": 3215
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 3220,
                "end": 3236
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3220,
              "end": 3236
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 3241,
                "end": 3251
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3241,
              "end": 3251
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 3256,
                "end": 3268
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3256,
              "end": 3268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 3273,
                "end": 3285
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3273,
              "end": 3285
            }
          }
        ],
        "loc": {
          "start": 3026,
          "end": 3287
        }
      },
      "loc": {
        "start": 3017,
        "end": 3287
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3288,
          "end": 3290
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3288,
        "end": 3290
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3291,
          "end": 3301
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3291,
        "end": 3301
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3302,
          "end": 3312
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3302,
        "end": 3312
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3313,
          "end": 3322
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3313,
        "end": 3322
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 3323,
          "end": 3334
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3323,
        "end": 3334
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 3335,
          "end": 3341
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 3351,
                "end": 3361
              }
            },
            "directives": [],
            "loc": {
              "start": 3348,
              "end": 3361
            }
          }
        ],
        "loc": {
          "start": 3342,
          "end": 3363
        }
      },
      "loc": {
        "start": 3335,
        "end": 3363
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 3364,
          "end": 3369
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 3383,
                  "end": 3387
                }
              },
              "loc": {
                "start": 3383,
                "end": 3387
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 3401,
                      "end": 3409
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3398,
                    "end": 3409
                  }
                }
              ],
              "loc": {
                "start": 3388,
                "end": 3415
              }
            },
            "loc": {
              "start": 3376,
              "end": 3415
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 3427,
                  "end": 3431
                }
              },
              "loc": {
                "start": 3427,
                "end": 3431
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 3445,
                      "end": 3453
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3442,
                    "end": 3453
                  }
                }
              ],
              "loc": {
                "start": 3432,
                "end": 3459
              }
            },
            "loc": {
              "start": 3420,
              "end": 3459
            }
          }
        ],
        "loc": {
          "start": 3370,
          "end": 3461
        }
      },
      "loc": {
        "start": 3364,
        "end": 3461
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 3462,
          "end": 3473
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3462,
        "end": 3473
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 3474,
          "end": 3488
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3474,
        "end": 3488
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 3489,
          "end": 3494
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3489,
        "end": 3494
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 3495,
          "end": 3504
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3495,
        "end": 3504
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 3505,
          "end": 3509
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 3519,
                "end": 3527
              }
            },
            "directives": [],
            "loc": {
              "start": 3516,
              "end": 3527
            }
          }
        ],
        "loc": {
          "start": 3510,
          "end": 3529
        }
      },
      "loc": {
        "start": 3505,
        "end": 3529
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 3530,
          "end": 3544
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3530,
        "end": 3544
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 3545,
          "end": 3550
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3545,
        "end": 3550
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 3551,
          "end": 3554
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 3561,
                "end": 3570
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3561,
              "end": 3570
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 3575,
                "end": 3586
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3575,
              "end": 3586
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 3591,
                "end": 3602
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3591,
              "end": 3602
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 3607,
                "end": 3616
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3607,
              "end": 3616
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 3621,
                "end": 3628
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3621,
              "end": 3628
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 3633,
                "end": 3641
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3633,
              "end": 3641
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 3646,
                "end": 3658
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3646,
              "end": 3658
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 3663,
                "end": 3671
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3663,
              "end": 3671
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 3676,
                "end": 3684
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3676,
              "end": 3684
            }
          }
        ],
        "loc": {
          "start": 3555,
          "end": 3686
        }
      },
      "loc": {
        "start": 3551,
        "end": 3686
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3723,
          "end": 3725
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3723,
        "end": 3725
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3726,
          "end": 3735
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3726,
        "end": 3735
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 3775,
          "end": 3787
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3794,
                "end": 3796
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3794,
              "end": 3796
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 3801,
                "end": 3809
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3801,
              "end": 3809
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 3814,
                "end": 3825
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3814,
              "end": 3825
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3830,
                "end": 3834
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3830,
              "end": 3834
            }
          }
        ],
        "loc": {
          "start": 3788,
          "end": 3836
        }
      },
      "loc": {
        "start": 3775,
        "end": 3836
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3837,
          "end": 3839
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3837,
        "end": 3839
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3840,
          "end": 3850
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3840,
        "end": 3850
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3851,
          "end": 3861
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3851,
        "end": 3861
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "createdBy",
        "loc": {
          "start": 3862,
          "end": 3871
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3878,
                "end": 3880
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3878,
              "end": 3880
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3885,
                "end": 3895
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3885,
              "end": 3895
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3900,
                "end": 3910
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3900,
              "end": 3910
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 3915,
                "end": 3926
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3915,
              "end": 3926
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 3931,
                "end": 3937
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3931,
              "end": 3937
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 3942,
                "end": 3947
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3942,
              "end": 3947
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 3952,
                "end": 3972
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3952,
              "end": 3972
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3977,
                "end": 3981
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3977,
              "end": 3981
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 3986,
                "end": 3998
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3986,
              "end": 3998
            }
          }
        ],
        "loc": {
          "start": 3872,
          "end": 4000
        }
      },
      "loc": {
        "start": 3862,
        "end": 4000
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "hasAcceptedAnswer",
        "loc": {
          "start": 4001,
          "end": 4018
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4001,
        "end": 4018
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 4019,
          "end": 4028
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4019,
        "end": 4028
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 4029,
          "end": 4034
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4029,
        "end": 4034
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 4035,
          "end": 4044
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4035,
        "end": 4044
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "answersCount",
        "loc": {
          "start": 4045,
          "end": 4057
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4045,
        "end": 4057
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 4058,
          "end": 4071
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4058,
        "end": 4071
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 4072,
          "end": 4084
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4072,
        "end": 4084
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "forObject",
        "loc": {
          "start": 4085,
          "end": 4094
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Api",
                "loc": {
                  "start": 4108,
                  "end": 4111
                }
              },
              "loc": {
                "start": 4108,
                "end": 4111
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Api_nav",
                    "loc": {
                      "start": 4125,
                      "end": 4132
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4122,
                    "end": 4132
                  }
                }
              ],
              "loc": {
                "start": 4112,
                "end": 4138
              }
            },
            "loc": {
              "start": 4101,
              "end": 4138
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Code",
                "loc": {
                  "start": 4150,
                  "end": 4154
                }
              },
              "loc": {
                "start": 4150,
                "end": 4154
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Code_nav",
                    "loc": {
                      "start": 4168,
                      "end": 4176
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4165,
                    "end": 4176
                  }
                }
              ],
              "loc": {
                "start": 4155,
                "end": 4182
              }
            },
            "loc": {
              "start": 4143,
              "end": 4182
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Note",
                "loc": {
                  "start": 4194,
                  "end": 4198
                }
              },
              "loc": {
                "start": 4194,
                "end": 4198
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Note_nav",
                    "loc": {
                      "start": 4212,
                      "end": 4220
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4209,
                    "end": 4220
                  }
                }
              ],
              "loc": {
                "start": 4199,
                "end": 4226
              }
            },
            "loc": {
              "start": 4187,
              "end": 4226
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Project",
                "loc": {
                  "start": 4238,
                  "end": 4245
                }
              },
              "loc": {
                "start": 4238,
                "end": 4245
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Project_nav",
                    "loc": {
                      "start": 4259,
                      "end": 4270
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4256,
                    "end": 4270
                  }
                }
              ],
              "loc": {
                "start": 4246,
                "end": 4276
              }
            },
            "loc": {
              "start": 4231,
              "end": 4276
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Routine",
                "loc": {
                  "start": 4288,
                  "end": 4295
                }
              },
              "loc": {
                "start": 4288,
                "end": 4295
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Routine_nav",
                    "loc": {
                      "start": 4309,
                      "end": 4320
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4306,
                    "end": 4320
                  }
                }
              ],
              "loc": {
                "start": 4296,
                "end": 4326
              }
            },
            "loc": {
              "start": 4281,
              "end": 4326
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Standard",
                "loc": {
                  "start": 4338,
                  "end": 4346
                }
              },
              "loc": {
                "start": 4338,
                "end": 4346
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Standard_nav",
                    "loc": {
                      "start": 4360,
                      "end": 4372
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4357,
                    "end": 4372
                  }
                }
              ],
              "loc": {
                "start": 4347,
                "end": 4378
              }
            },
            "loc": {
              "start": 4331,
              "end": 4378
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 4390,
                  "end": 4394
                }
              },
              "loc": {
                "start": 4390,
                "end": 4394
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 4408,
                      "end": 4416
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4405,
                    "end": 4416
                  }
                }
              ],
              "loc": {
                "start": 4395,
                "end": 4422
              }
            },
            "loc": {
              "start": 4383,
              "end": 4422
            }
          }
        ],
        "loc": {
          "start": 4095,
          "end": 4424
        }
      },
      "loc": {
        "start": 4085,
        "end": 4424
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 4425,
          "end": 4429
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 4439,
                "end": 4447
              }
            },
            "directives": [],
            "loc": {
              "start": 4436,
              "end": 4447
            }
          }
        ],
        "loc": {
          "start": 4430,
          "end": 4449
        }
      },
      "loc": {
        "start": 4425,
        "end": 4449
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 4450,
          "end": 4453
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 4460,
                "end": 4468
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4460,
              "end": 4468
            }
          }
        ],
        "loc": {
          "start": 4454,
          "end": 4470
        }
      },
      "loc": {
        "start": 4450,
        "end": 4470
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 4508,
          "end": 4516
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 4523,
                "end": 4535
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4546,
                      "end": 4548
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4546,
                    "end": 4548
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 4557,
                      "end": 4565
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4557,
                    "end": 4565
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 4574,
                      "end": 4585
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4574,
                    "end": 4585
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 4594,
                      "end": 4606
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4594,
                    "end": 4606
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4615,
                      "end": 4619
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4615,
                    "end": 4619
                  }
                }
              ],
              "loc": {
                "start": 4536,
                "end": 4625
              }
            },
            "loc": {
              "start": 4523,
              "end": 4625
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 4630,
                "end": 4632
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4630,
              "end": 4632
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4637,
                "end": 4647
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4637,
              "end": 4647
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4652,
                "end": 4662
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4652,
              "end": 4662
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 4667,
                "end": 4678
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4667,
              "end": 4678
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 4683,
                "end": 4696
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4683,
              "end": 4696
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 4701,
                "end": 4711
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4701,
              "end": 4711
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 4716,
                "end": 4725
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4716,
              "end": 4725
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 4730,
                "end": 4738
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4730,
              "end": 4738
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4743,
                "end": 4752
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4743,
              "end": 4752
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routineType",
              "loc": {
                "start": 4757,
                "end": 4768
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4757,
              "end": 4768
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 4773,
                "end": 4783
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4773,
              "end": 4783
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesStarted",
              "loc": {
                "start": 4788,
                "end": 4800
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4788,
              "end": 4800
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesCompleted",
              "loc": {
                "start": 4805,
                "end": 4819
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4805,
              "end": 4819
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 4824,
                "end": 4836
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4824,
              "end": 4836
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 4841,
                "end": 4853
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4841,
              "end": 4853
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 4858,
                "end": 4871
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4858,
              "end": 4871
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 4876,
                "end": 4898
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4876,
              "end": 4898
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 4903,
                "end": 4913
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4903,
              "end": 4913
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 4918,
                "end": 4929
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4918,
              "end": 4929
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodesCount",
              "loc": {
                "start": 4934,
                "end": 4944
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4934,
              "end": 4944
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodeLinksCount",
              "loc": {
                "start": 4949,
                "end": 4963
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4949,
              "end": 4963
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
              "loc": {
                "start": 4968,
                "end": 4980
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4968,
              "end": 4980
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 4985,
                "end": 4997
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4985,
              "end": 4997
            }
          }
        ],
        "loc": {
          "start": 4517,
          "end": 4999
        }
      },
      "loc": {
        "start": 4508,
        "end": 4999
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5000,
          "end": 5002
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5000,
        "end": 5002
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 5003,
          "end": 5013
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5003,
        "end": 5013
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 5014,
          "end": 5024
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5014,
        "end": 5024
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 5025,
          "end": 5035
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5025,
        "end": 5035
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5036,
          "end": 5045
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5036,
        "end": 5045
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 5046,
          "end": 5057
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5046,
        "end": 5057
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 5058,
          "end": 5064
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 5074,
                "end": 5084
              }
            },
            "directives": [],
            "loc": {
              "start": 5071,
              "end": 5084
            }
          }
        ],
        "loc": {
          "start": 5065,
          "end": 5086
        }
      },
      "loc": {
        "start": 5058,
        "end": 5086
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 5087,
          "end": 5092
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 5106,
                  "end": 5110
                }
              },
              "loc": {
                "start": 5106,
                "end": 5110
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 5124,
                      "end": 5132
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5121,
                    "end": 5132
                  }
                }
              ],
              "loc": {
                "start": 5111,
                "end": 5138
              }
            },
            "loc": {
              "start": 5099,
              "end": 5138
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 5150,
                  "end": 5154
                }
              },
              "loc": {
                "start": 5150,
                "end": 5154
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 5168,
                      "end": 5176
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5165,
                    "end": 5176
                  }
                }
              ],
              "loc": {
                "start": 5155,
                "end": 5182
              }
            },
            "loc": {
              "start": 5143,
              "end": 5182
            }
          }
        ],
        "loc": {
          "start": 5093,
          "end": 5184
        }
      },
      "loc": {
        "start": 5087,
        "end": 5184
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 5185,
          "end": 5196
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5185,
        "end": 5196
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 5197,
          "end": 5211
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5197,
        "end": 5211
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 5212,
          "end": 5217
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5212,
        "end": 5217
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 5218,
          "end": 5227
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5218,
        "end": 5227
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 5228,
          "end": 5232
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 5242,
                "end": 5250
              }
            },
            "directives": [],
            "loc": {
              "start": 5239,
              "end": 5250
            }
          }
        ],
        "loc": {
          "start": 5233,
          "end": 5252
        }
      },
      "loc": {
        "start": 5228,
        "end": 5252
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 5253,
          "end": 5267
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5253,
        "end": 5267
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 5268,
          "end": 5273
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5268,
        "end": 5273
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 5274,
          "end": 5277
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canComment",
              "loc": {
                "start": 5284,
                "end": 5294
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5284,
              "end": 5294
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 5299,
                "end": 5308
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5299,
              "end": 5308
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 5313,
                "end": 5324
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5313,
              "end": 5324
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 5329,
                "end": 5338
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5329,
              "end": 5338
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 5343,
                "end": 5350
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5343,
              "end": 5350
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 5355,
                "end": 5363
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5355,
              "end": 5363
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 5368,
                "end": 5380
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5368,
              "end": 5380
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 5385,
                "end": 5393
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5385,
              "end": 5393
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 5398,
                "end": 5406
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5398,
              "end": 5406
            }
          }
        ],
        "loc": {
          "start": 5278,
          "end": 5408
        }
      },
      "loc": {
        "start": 5274,
        "end": 5408
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5445,
          "end": 5447
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5445,
        "end": 5447
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 5448,
          "end": 5458
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5448,
        "end": 5458
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5459,
          "end": 5468
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5459,
        "end": 5468
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 5508,
          "end": 5516
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 5523,
                "end": 5535
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5546,
                      "end": 5548
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5546,
                    "end": 5548
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 5557,
                      "end": 5565
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5557,
                    "end": 5565
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 5574,
                      "end": 5585
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5574,
                    "end": 5585
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 5594,
                      "end": 5606
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5594,
                    "end": 5606
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5615,
                      "end": 5619
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5615,
                    "end": 5619
                  }
                }
              ],
              "loc": {
                "start": 5536,
                "end": 5625
              }
            },
            "loc": {
              "start": 5523,
              "end": 5625
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5630,
                "end": 5632
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5630,
              "end": 5632
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5637,
                "end": 5647
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5637,
              "end": 5647
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5652,
                "end": 5662
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5652,
              "end": 5662
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 5667,
                "end": 5677
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5667,
              "end": 5677
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isFile",
              "loc": {
                "start": 5682,
                "end": 5688
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5682,
              "end": 5688
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 5693,
                "end": 5701
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5693,
              "end": 5701
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5706,
                "end": 5715
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5706,
              "end": 5715
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 5720,
                "end": 5727
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5720,
              "end": 5727
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardType",
              "loc": {
                "start": 5732,
                "end": 5744
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5732,
              "end": 5744
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "props",
              "loc": {
                "start": 5749,
                "end": 5754
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5749,
              "end": 5754
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yup",
              "loc": {
                "start": 5759,
                "end": 5762
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5759,
              "end": 5762
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 5767,
                "end": 5779
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5767,
              "end": 5779
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 5784,
                "end": 5796
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5784,
              "end": 5796
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 5801,
                "end": 5814
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5801,
              "end": 5814
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 5819,
                "end": 5841
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5819,
              "end": 5841
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 5846,
                "end": 5856
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5846,
              "end": 5856
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 5861,
                "end": 5873
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5861,
              "end": 5873
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5878,
                "end": 5881
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 5892,
                      "end": 5902
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5892,
                    "end": 5902
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 5911,
                      "end": 5918
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5911,
                    "end": 5918
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5927,
                      "end": 5936
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5927,
                    "end": 5936
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 5945,
                      "end": 5954
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5945,
                    "end": 5954
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5963,
                      "end": 5972
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5963,
                    "end": 5972
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 5981,
                      "end": 5987
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5981,
                    "end": 5987
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5996,
                      "end": 6003
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5996,
                    "end": 6003
                  }
                }
              ],
              "loc": {
                "start": 5882,
                "end": 6009
              }
            },
            "loc": {
              "start": 5878,
              "end": 6009
            }
          }
        ],
        "loc": {
          "start": 5517,
          "end": 6011
        }
      },
      "loc": {
        "start": 5508,
        "end": 6011
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6012,
          "end": 6014
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6012,
        "end": 6014
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6015,
          "end": 6025
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6015,
        "end": 6025
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 6026,
          "end": 6036
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6026,
        "end": 6036
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6037,
          "end": 6046
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6037,
        "end": 6046
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 6047,
          "end": 6058
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6047,
        "end": 6058
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 6059,
          "end": 6065
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 6075,
                "end": 6085
              }
            },
            "directives": [],
            "loc": {
              "start": 6072,
              "end": 6085
            }
          }
        ],
        "loc": {
          "start": 6066,
          "end": 6087
        }
      },
      "loc": {
        "start": 6059,
        "end": 6087
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 6088,
          "end": 6093
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 6107,
                  "end": 6111
                }
              },
              "loc": {
                "start": 6107,
                "end": 6111
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 6125,
                      "end": 6133
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6122,
                    "end": 6133
                  }
                }
              ],
              "loc": {
                "start": 6112,
                "end": 6139
              }
            },
            "loc": {
              "start": 6100,
              "end": 6139
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 6151,
                  "end": 6155
                }
              },
              "loc": {
                "start": 6151,
                "end": 6155
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 6169,
                      "end": 6177
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6166,
                    "end": 6177
                  }
                }
              ],
              "loc": {
                "start": 6156,
                "end": 6183
              }
            },
            "loc": {
              "start": 6144,
              "end": 6183
            }
          }
        ],
        "loc": {
          "start": 6094,
          "end": 6185
        }
      },
      "loc": {
        "start": 6088,
        "end": 6185
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 6186,
          "end": 6197
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6186,
        "end": 6197
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 6198,
          "end": 6212
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6198,
        "end": 6212
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 6213,
          "end": 6218
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6213,
        "end": 6218
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6219,
          "end": 6228
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6219,
        "end": 6228
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 6229,
          "end": 6233
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 6243,
                "end": 6251
              }
            },
            "directives": [],
            "loc": {
              "start": 6240,
              "end": 6251
            }
          }
        ],
        "loc": {
          "start": 6234,
          "end": 6253
        }
      },
      "loc": {
        "start": 6229,
        "end": 6253
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 6254,
          "end": 6268
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6254,
        "end": 6268
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 6269,
          "end": 6274
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6269,
        "end": 6274
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6275,
          "end": 6278
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 6285,
                "end": 6294
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6285,
              "end": 6294
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 6299,
                "end": 6310
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6299,
              "end": 6310
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 6315,
                "end": 6326
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6315,
              "end": 6326
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 6331,
                "end": 6340
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6331,
              "end": 6340
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 6345,
                "end": 6352
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6345,
              "end": 6352
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 6357,
                "end": 6365
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6357,
              "end": 6365
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6370,
                "end": 6382
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6370,
              "end": 6382
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 6387,
                "end": 6395
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6387,
              "end": 6395
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 6400,
                "end": 6408
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6400,
              "end": 6408
            }
          }
        ],
        "loc": {
          "start": 6279,
          "end": 6410
        }
      },
      "loc": {
        "start": 6275,
        "end": 6410
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6449,
          "end": 6451
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6449,
        "end": 6451
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6452,
          "end": 6461
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6452,
        "end": 6461
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6491,
          "end": 6493
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6491,
        "end": 6493
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6494,
          "end": 6504
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6494,
        "end": 6504
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 6505,
          "end": 6508
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6505,
        "end": 6508
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6509,
          "end": 6518
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6509,
        "end": 6518
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 6519,
          "end": 6531
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6538,
                "end": 6540
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6538,
              "end": 6540
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 6545,
                "end": 6553
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6545,
              "end": 6553
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 6558,
                "end": 6569
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6558,
              "end": 6569
            }
          }
        ],
        "loc": {
          "start": 6532,
          "end": 6571
        }
      },
      "loc": {
        "start": 6519,
        "end": 6571
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6572,
          "end": 6575
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOwn",
              "loc": {
                "start": 6582,
                "end": 6587
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6582,
              "end": 6587
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6592,
                "end": 6604
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6592,
              "end": 6604
            }
          }
        ],
        "loc": {
          "start": 6576,
          "end": 6606
        }
      },
      "loc": {
        "start": 6572,
        "end": 6606
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6638,
          "end": 6640
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6638,
        "end": 6640
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 6641,
          "end": 6652
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6641,
        "end": 6652
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 6653,
          "end": 6659
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6653,
        "end": 6659
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6660,
          "end": 6670
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6660,
        "end": 6670
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 6671,
          "end": 6681
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6671,
        "end": 6681
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isOpenToNewMembers",
        "loc": {
          "start": 6682,
          "end": 6700
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6682,
        "end": 6700
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6701,
          "end": 6710
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6701,
        "end": 6710
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 6711,
          "end": 6724
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6711,
        "end": 6724
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "membersCount",
        "loc": {
          "start": 6725,
          "end": 6737
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6725,
        "end": 6737
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 6738,
          "end": 6750
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6738,
        "end": 6750
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 6751,
          "end": 6763
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6751,
        "end": 6763
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6764,
          "end": 6773
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6764,
        "end": 6773
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 6774,
          "end": 6778
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 6788,
                "end": 6796
              }
            },
            "directives": [],
            "loc": {
              "start": 6785,
              "end": 6796
            }
          }
        ],
        "loc": {
          "start": 6779,
          "end": 6798
        }
      },
      "loc": {
        "start": 6774,
        "end": 6798
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 6799,
          "end": 6811
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6818,
                "end": 6820
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6818,
              "end": 6820
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 6825,
                "end": 6833
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6825,
              "end": 6833
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 6838,
                "end": 6841
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6838,
              "end": 6841
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 6846,
                "end": 6850
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6846,
              "end": 6850
            }
          }
        ],
        "loc": {
          "start": 6812,
          "end": 6852
        }
      },
      "loc": {
        "start": 6799,
        "end": 6852
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6853,
          "end": 6856
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 6863,
                "end": 6876
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6863,
              "end": 6876
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 6881,
                "end": 6890
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6881,
              "end": 6890
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 6895,
                "end": 6906
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6895,
              "end": 6906
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 6911,
                "end": 6920
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6911,
              "end": 6920
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 6925,
                "end": 6934
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6925,
              "end": 6934
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 6939,
                "end": 6946
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6939,
              "end": 6946
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6951,
                "end": 6963
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6951,
              "end": 6963
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 6968,
                "end": 6976
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6968,
              "end": 6976
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 6981,
                "end": 6995
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7006,
                      "end": 7008
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7006,
                    "end": 7008
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 7017,
                      "end": 7027
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7017,
                    "end": 7027
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 7036,
                      "end": 7046
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7036,
                    "end": 7046
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 7055,
                      "end": 7062
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7055,
                    "end": 7062
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 7071,
                      "end": 7082
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7071,
                    "end": 7082
                  }
                }
              ],
              "loc": {
                "start": 6996,
                "end": 7088
              }
            },
            "loc": {
              "start": 6981,
              "end": 7088
            }
          }
        ],
        "loc": {
          "start": 6857,
          "end": 7090
        }
      },
      "loc": {
        "start": 6853,
        "end": 7090
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7121,
          "end": 7123
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7121,
        "end": 7123
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7124,
          "end": 7135
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7124,
        "end": 7135
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7136,
          "end": 7142
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7136,
        "end": 7142
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7143,
          "end": 7155
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7143,
        "end": 7155
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7156,
          "end": 7159
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 7166,
                "end": 7179
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7166,
              "end": 7179
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 7184,
                "end": 7193
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7184,
              "end": 7193
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 7198,
                "end": 7209
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7198,
              "end": 7209
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 7214,
                "end": 7223
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7214,
              "end": 7223
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7228,
                "end": 7237
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7228,
              "end": 7237
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 7242,
                "end": 7249
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7242,
              "end": 7249
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7254,
                "end": 7266
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7254,
              "end": 7266
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7271,
                "end": 7279
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7271,
              "end": 7279
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 7284,
                "end": 7298
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7309,
                      "end": 7311
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7309,
                    "end": 7311
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 7320,
                      "end": 7330
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7320,
                    "end": 7330
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 7339,
                      "end": 7349
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7339,
                    "end": 7349
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 7358,
                      "end": 7365
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7358,
                    "end": 7365
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 7374,
                      "end": 7385
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7374,
                    "end": 7385
                  }
                }
              ],
              "loc": {
                "start": 7299,
                "end": 7391
              }
            },
            "loc": {
              "start": 7284,
              "end": 7391
            }
          }
        ],
        "loc": {
          "start": 7160,
          "end": 7393
        }
      },
      "loc": {
        "start": 7156,
        "end": 7393
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 7425,
          "end": 7437
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7444,
                "end": 7446
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7444,
              "end": 7446
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 7451,
                "end": 7459
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7451,
              "end": 7459
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 7464,
                "end": 7467
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7464,
              "end": 7467
            }
          }
        ],
        "loc": {
          "start": 7438,
          "end": 7469
        }
      },
      "loc": {
        "start": 7425,
        "end": 7469
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7470,
          "end": 7472
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7470,
        "end": 7472
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7473,
          "end": 7483
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7473,
        "end": 7483
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7484,
          "end": 7494
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7484,
        "end": 7494
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7495,
          "end": 7506
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7495,
        "end": 7506
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7507,
          "end": 7513
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7507,
        "end": 7513
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7514,
          "end": 7519
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7514,
        "end": 7519
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 7520,
          "end": 7540
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7520,
        "end": 7540
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7541,
          "end": 7545
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7541,
        "end": 7545
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7546,
          "end": 7558
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7546,
        "end": 7558
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7559,
          "end": 7568
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7559,
        "end": 7568
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsReceivedCount",
        "loc": {
          "start": 7569,
          "end": 7589
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7569,
        "end": 7589
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7590,
          "end": 7593
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 7600,
                "end": 7609
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7600,
              "end": 7609
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 7614,
                "end": 7623
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7614,
              "end": 7623
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7628,
                "end": 7637
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7628,
              "end": 7637
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7642,
                "end": 7654
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7642,
              "end": 7654
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7659,
                "end": 7667
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7659,
              "end": 7667
            }
          }
        ],
        "loc": {
          "start": 7594,
          "end": 7669
        }
      },
      "loc": {
        "start": 7590,
        "end": 7669
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7700,
          "end": 7702
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7700,
        "end": 7702
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7703,
          "end": 7713
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7703,
        "end": 7713
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7714,
          "end": 7724
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7714,
        "end": 7724
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7725,
          "end": 7736
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7725,
        "end": 7736
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7737,
          "end": 7743
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7737,
        "end": 7743
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7744,
          "end": 7749
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7744,
        "end": 7749
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 7750,
          "end": 7770
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7750,
        "end": 7770
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7771,
          "end": 7775
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7771,
        "end": 7775
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7776,
          "end": 7788
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7776,
        "end": 7788
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Api_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Api_list",
        "loc": {
          "start": 10,
          "end": 18
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 22,
            "end": 25
          }
        },
        "loc": {
          "start": 22,
          "end": 25
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 28,
                "end": 36
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 43,
                      "end": 55
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 66,
                            "end": 68
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 66,
                          "end": 68
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 77,
                            "end": 85
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 77,
                          "end": 85
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "details",
                          "loc": {
                            "start": 94,
                            "end": 101
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 94,
                          "end": 101
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 110,
                            "end": 114
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 110,
                          "end": 114
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "summary",
                          "loc": {
                            "start": 123,
                            "end": 130
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 123,
                          "end": 130
                        }
                      }
                    ],
                    "loc": {
                      "start": 56,
                      "end": 136
                    }
                  },
                  "loc": {
                    "start": 43,
                    "end": 136
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 141,
                      "end": 143
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 141,
                    "end": 143
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 148,
                      "end": 158
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 148,
                    "end": 158
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 163,
                      "end": 173
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 163,
                    "end": 173
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "callLink",
                    "loc": {
                      "start": 178,
                      "end": 186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 178,
                    "end": 186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 191,
                      "end": 204
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 191,
                    "end": 204
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "documentationLink",
                    "loc": {
                      "start": 209,
                      "end": 226
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 209,
                    "end": 226
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 231,
                      "end": 241
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 231,
                    "end": 241
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 246,
                      "end": 254
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 246,
                    "end": 254
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 259,
                      "end": 268
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 259,
                    "end": 268
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 273,
                      "end": 285
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 273,
                    "end": 285
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 290,
                      "end": 302
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 290,
                    "end": 302
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 307,
                      "end": 319
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 307,
                    "end": 319
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 324,
                      "end": 327
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 338,
                            "end": 348
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 338,
                          "end": 348
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 357,
                            "end": 364
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 357,
                          "end": 364
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 373,
                            "end": 382
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 373,
                          "end": 382
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 391,
                            "end": 400
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 391,
                          "end": 400
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 409,
                            "end": 418
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 409,
                          "end": 418
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 427,
                            "end": 433
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 427,
                          "end": 433
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 442,
                            "end": 449
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 442,
                          "end": 449
                        }
                      }
                    ],
                    "loc": {
                      "start": 328,
                      "end": 455
                    }
                  },
                  "loc": {
                    "start": 324,
                    "end": 455
                  }
                }
              ],
              "loc": {
                "start": 37,
                "end": 457
              }
            },
            "loc": {
              "start": 28,
              "end": 457
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 458,
                "end": 460
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 458,
              "end": 460
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 461,
                "end": 471
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 461,
              "end": 471
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 472,
                "end": 482
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 472,
              "end": 482
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 483,
                "end": 492
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 483,
              "end": 492
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 493,
                "end": 504
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 493,
              "end": 504
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 505,
                "end": 511
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 521,
                      "end": 531
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 518,
                    "end": 531
                  }
                }
              ],
              "loc": {
                "start": 512,
                "end": 533
              }
            },
            "loc": {
              "start": 505,
              "end": 533
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 534,
                "end": 539
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 553,
                        "end": 557
                      }
                    },
                    "loc": {
                      "start": 553,
                      "end": 557
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 571,
                            "end": 579
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 568,
                          "end": 579
                        }
                      }
                    ],
                    "loc": {
                      "start": 558,
                      "end": 585
                    }
                  },
                  "loc": {
                    "start": 546,
                    "end": 585
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 597,
                        "end": 601
                      }
                    },
                    "loc": {
                      "start": 597,
                      "end": 601
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 615,
                            "end": 623
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 612,
                          "end": 623
                        }
                      }
                    ],
                    "loc": {
                      "start": 602,
                      "end": 629
                    }
                  },
                  "loc": {
                    "start": 590,
                    "end": 629
                  }
                }
              ],
              "loc": {
                "start": 540,
                "end": 631
              }
            },
            "loc": {
              "start": 534,
              "end": 631
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 632,
                "end": 643
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 632,
              "end": 643
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 644,
                "end": 658
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 644,
              "end": 658
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 659,
                "end": 664
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 659,
              "end": 664
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 665,
                "end": 674
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 665,
              "end": 674
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 675,
                "end": 679
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 689,
                      "end": 697
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 686,
                    "end": 697
                  }
                }
              ],
              "loc": {
                "start": 680,
                "end": 699
              }
            },
            "loc": {
              "start": 675,
              "end": 699
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 700,
                "end": 714
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 700,
              "end": 714
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 715,
                "end": 720
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 715,
              "end": 720
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 721,
                "end": 724
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 731,
                      "end": 740
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 731,
                    "end": 740
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 745,
                      "end": 756
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 745,
                    "end": 756
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 761,
                      "end": 772
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 761,
                    "end": 772
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 777,
                      "end": 786
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 777,
                    "end": 786
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 791,
                      "end": 798
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 791,
                    "end": 798
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 803,
                      "end": 811
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 803,
                    "end": 811
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 816,
                      "end": 828
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 816,
                    "end": 828
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 833,
                      "end": 841
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 833,
                    "end": 841
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 846,
                      "end": 854
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 846,
                    "end": 854
                  }
                }
              ],
              "loc": {
                "start": 725,
                "end": 856
              }
            },
            "loc": {
              "start": 721,
              "end": 856
            }
          }
        ],
        "loc": {
          "start": 26,
          "end": 858
        }
      },
      "loc": {
        "start": 1,
        "end": 858
      }
    },
    "Api_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Api_nav",
        "loc": {
          "start": 868,
          "end": 875
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 879,
            "end": 882
          }
        },
        "loc": {
          "start": 879,
          "end": 882
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 885,
                "end": 887
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 885,
              "end": 887
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 888,
                "end": 897
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 888,
              "end": 897
            }
          }
        ],
        "loc": {
          "start": 883,
          "end": 899
        }
      },
      "loc": {
        "start": 859,
        "end": 899
      }
    },
    "Code_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Code_list",
        "loc": {
          "start": 909,
          "end": 918
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Code",
          "loc": {
            "start": 922,
            "end": 926
          }
        },
        "loc": {
          "start": 922,
          "end": 926
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 929,
                "end": 937
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 944,
                      "end": 956
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 967,
                            "end": 969
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 967,
                          "end": 969
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 978,
                            "end": 986
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 978,
                          "end": 986
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 995,
                            "end": 1006
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 995,
                          "end": 1006
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 1015,
                            "end": 1027
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1015,
                          "end": 1027
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1036,
                            "end": 1040
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1036,
                          "end": 1040
                        }
                      }
                    ],
                    "loc": {
                      "start": 957,
                      "end": 1046
                    }
                  },
                  "loc": {
                    "start": 944,
                    "end": 1046
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1051,
                      "end": 1053
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1051,
                    "end": 1053
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1058,
                      "end": 1068
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1058,
                    "end": 1068
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1073,
                      "end": 1083
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1073,
                    "end": 1083
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1088,
                      "end": 1098
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1088,
                    "end": 1098
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 1103,
                      "end": 1112
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1103,
                    "end": 1112
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1117,
                      "end": 1125
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1117,
                    "end": 1125
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1130,
                      "end": 1139
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1130,
                    "end": 1139
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "codeLanguage",
                    "loc": {
                      "start": 1144,
                      "end": 1156
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1144,
                    "end": 1156
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "codeType",
                    "loc": {
                      "start": 1161,
                      "end": 1169
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1161,
                    "end": 1169
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 1174,
                      "end": 1181
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1174,
                    "end": 1181
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1186,
                      "end": 1198
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1186,
                    "end": 1198
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1203,
                      "end": 1215
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1203,
                    "end": 1215
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 1220,
                      "end": 1233
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1220,
                    "end": 1233
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 1238,
                      "end": 1260
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1238,
                    "end": 1260
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 1265,
                      "end": 1275
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1265,
                    "end": 1275
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1280,
                      "end": 1292
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1280,
                    "end": 1292
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 1297,
                      "end": 1300
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 1311,
                            "end": 1321
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1311,
                          "end": 1321
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 1330,
                            "end": 1337
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1330,
                          "end": 1337
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 1346,
                            "end": 1355
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1346,
                          "end": 1355
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 1364,
                            "end": 1373
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1364,
                          "end": 1373
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 1382,
                            "end": 1391
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1382,
                          "end": 1391
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 1400,
                            "end": 1406
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1400,
                          "end": 1406
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 1415,
                            "end": 1422
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1415,
                          "end": 1422
                        }
                      }
                    ],
                    "loc": {
                      "start": 1301,
                      "end": 1428
                    }
                  },
                  "loc": {
                    "start": 1297,
                    "end": 1428
                  }
                }
              ],
              "loc": {
                "start": 938,
                "end": 1430
              }
            },
            "loc": {
              "start": 929,
              "end": 1430
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1431,
                "end": 1433
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1431,
              "end": 1433
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1434,
                "end": 1444
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1434,
              "end": 1444
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1445,
                "end": 1455
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1445,
              "end": 1455
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1456,
                "end": 1465
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1456,
              "end": 1465
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1466,
                "end": 1477
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1466,
              "end": 1477
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1478,
                "end": 1484
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 1494,
                      "end": 1504
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1491,
                    "end": 1504
                  }
                }
              ],
              "loc": {
                "start": 1485,
                "end": 1506
              }
            },
            "loc": {
              "start": 1478,
              "end": 1506
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1507,
                "end": 1512
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 1526,
                        "end": 1530
                      }
                    },
                    "loc": {
                      "start": 1526,
                      "end": 1530
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 1544,
                            "end": 1552
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1541,
                          "end": 1552
                        }
                      }
                    ],
                    "loc": {
                      "start": 1531,
                      "end": 1558
                    }
                  },
                  "loc": {
                    "start": 1519,
                    "end": 1558
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 1570,
                        "end": 1574
                      }
                    },
                    "loc": {
                      "start": 1570,
                      "end": 1574
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 1588,
                            "end": 1596
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1585,
                          "end": 1596
                        }
                      }
                    ],
                    "loc": {
                      "start": 1575,
                      "end": 1602
                    }
                  },
                  "loc": {
                    "start": 1563,
                    "end": 1602
                  }
                }
              ],
              "loc": {
                "start": 1513,
                "end": 1604
              }
            },
            "loc": {
              "start": 1507,
              "end": 1604
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1605,
                "end": 1616
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1605,
              "end": 1616
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1617,
                "end": 1631
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1617,
              "end": 1631
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1632,
                "end": 1637
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1632,
              "end": 1637
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1638,
                "end": 1647
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1638,
              "end": 1647
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1648,
                "end": 1652
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 1662,
                      "end": 1670
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1659,
                    "end": 1670
                  }
                }
              ],
              "loc": {
                "start": 1653,
                "end": 1672
              }
            },
            "loc": {
              "start": 1648,
              "end": 1672
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1673,
                "end": 1687
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1673,
              "end": 1687
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1688,
                "end": 1693
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1688,
              "end": 1693
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1694,
                "end": 1697
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1704,
                      "end": 1713
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1704,
                    "end": 1713
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1718,
                      "end": 1729
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1718,
                    "end": 1729
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 1734,
                      "end": 1745
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1734,
                    "end": 1745
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1750,
                      "end": 1759
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1750,
                    "end": 1759
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1764,
                      "end": 1771
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1764,
                    "end": 1771
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1776,
                      "end": 1784
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1776,
                    "end": 1784
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1789,
                      "end": 1801
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1789,
                    "end": 1801
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1806,
                      "end": 1814
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1806,
                    "end": 1814
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1819,
                      "end": 1827
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1819,
                    "end": 1827
                  }
                }
              ],
              "loc": {
                "start": 1698,
                "end": 1829
              }
            },
            "loc": {
              "start": 1694,
              "end": 1829
            }
          }
        ],
        "loc": {
          "start": 927,
          "end": 1831
        }
      },
      "loc": {
        "start": 900,
        "end": 1831
      }
    },
    "Code_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Code_nav",
        "loc": {
          "start": 1841,
          "end": 1849
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Code",
          "loc": {
            "start": 1853,
            "end": 1857
          }
        },
        "loc": {
          "start": 1853,
          "end": 1857
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1860,
                "end": 1862
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1860,
              "end": 1862
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1863,
                "end": 1872
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1863,
              "end": 1872
            }
          }
        ],
        "loc": {
          "start": 1858,
          "end": 1874
        }
      },
      "loc": {
        "start": 1832,
        "end": 1874
      }
    },
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 1884,
          "end": 1894
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 1898,
            "end": 1903
          }
        },
        "loc": {
          "start": 1898,
          "end": 1903
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1906,
                "end": 1908
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1906,
              "end": 1908
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1909,
                "end": 1919
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1909,
              "end": 1919
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1920,
                "end": 1930
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1920,
              "end": 1930
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 1931,
                "end": 1936
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1931,
              "end": 1936
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 1937,
                "end": 1942
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1937,
              "end": 1942
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1943,
                "end": 1948
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 1962,
                        "end": 1966
                      }
                    },
                    "loc": {
                      "start": 1962,
                      "end": 1966
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 1980,
                            "end": 1988
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1977,
                          "end": 1988
                        }
                      }
                    ],
                    "loc": {
                      "start": 1967,
                      "end": 1994
                    }
                  },
                  "loc": {
                    "start": 1955,
                    "end": 1994
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 2006,
                        "end": 2010
                      }
                    },
                    "loc": {
                      "start": 2006,
                      "end": 2010
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 2024,
                            "end": 2032
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2021,
                          "end": 2032
                        }
                      }
                    ],
                    "loc": {
                      "start": 2011,
                      "end": 2038
                    }
                  },
                  "loc": {
                    "start": 1999,
                    "end": 2038
                  }
                }
              ],
              "loc": {
                "start": 1949,
                "end": 2040
              }
            },
            "loc": {
              "start": 1943,
              "end": 2040
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2041,
                "end": 2044
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2051,
                      "end": 2060
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2051,
                    "end": 2060
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2065,
                      "end": 2074
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2065,
                    "end": 2074
                  }
                }
              ],
              "loc": {
                "start": 2045,
                "end": 2076
              }
            },
            "loc": {
              "start": 2041,
              "end": 2076
            }
          }
        ],
        "loc": {
          "start": 1904,
          "end": 2078
        }
      },
      "loc": {
        "start": 1875,
        "end": 2078
      }
    },
    "Note_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_list",
        "loc": {
          "start": 2088,
          "end": 2097
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 2101,
            "end": 2105
          }
        },
        "loc": {
          "start": 2101,
          "end": 2105
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 2108,
                "end": 2116
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 2123,
                      "end": 2135
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2146,
                            "end": 2148
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2146,
                          "end": 2148
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 2157,
                            "end": 2165
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2157,
                          "end": 2165
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2174,
                            "end": 2185
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2174,
                          "end": 2185
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2194,
                            "end": 2198
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2194,
                          "end": 2198
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pages",
                          "loc": {
                            "start": 2207,
                            "end": 2212
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 2227,
                                  "end": 2229
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2227,
                                "end": 2229
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "pageIndex",
                                "loc": {
                                  "start": 2242,
                                  "end": 2251
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2242,
                                "end": 2251
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "text",
                                "loc": {
                                  "start": 2264,
                                  "end": 2268
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2264,
                                "end": 2268
                              }
                            }
                          ],
                          "loc": {
                            "start": 2213,
                            "end": 2278
                          }
                        },
                        "loc": {
                          "start": 2207,
                          "end": 2278
                        }
                      }
                    ],
                    "loc": {
                      "start": 2136,
                      "end": 2284
                    }
                  },
                  "loc": {
                    "start": 2123,
                    "end": 2284
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2289,
                      "end": 2291
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2289,
                    "end": 2291
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2296,
                      "end": 2306
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2296,
                    "end": 2306
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2311,
                      "end": 2321
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2311,
                    "end": 2321
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 2326,
                      "end": 2334
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2326,
                    "end": 2334
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 2339,
                      "end": 2348
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2339,
                    "end": 2348
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 2353,
                      "end": 2365
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2353,
                    "end": 2365
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 2370,
                      "end": 2382
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2370,
                    "end": 2382
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 2387,
                      "end": 2399
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2387,
                    "end": 2399
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 2404,
                      "end": 2407
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 2418,
                            "end": 2428
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2418,
                          "end": 2428
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 2437,
                            "end": 2444
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2437,
                          "end": 2444
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 2453,
                            "end": 2462
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2453,
                          "end": 2462
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 2471,
                            "end": 2480
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2471,
                          "end": 2480
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 2489,
                            "end": 2498
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2489,
                          "end": 2498
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 2507,
                            "end": 2513
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2507,
                          "end": 2513
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 2522,
                            "end": 2529
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2522,
                          "end": 2529
                        }
                      }
                    ],
                    "loc": {
                      "start": 2408,
                      "end": 2535
                    }
                  },
                  "loc": {
                    "start": 2404,
                    "end": 2535
                  }
                }
              ],
              "loc": {
                "start": 2117,
                "end": 2537
              }
            },
            "loc": {
              "start": 2108,
              "end": 2537
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2538,
                "end": 2540
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2538,
              "end": 2540
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2541,
                "end": 2551
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2541,
              "end": 2551
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2552,
                "end": 2562
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2552,
              "end": 2562
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2563,
                "end": 2572
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2563,
              "end": 2572
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 2573,
                "end": 2584
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2573,
              "end": 2584
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 2585,
                "end": 2591
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 2601,
                      "end": 2611
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2598,
                    "end": 2611
                  }
                }
              ],
              "loc": {
                "start": 2592,
                "end": 2613
              }
            },
            "loc": {
              "start": 2585,
              "end": 2613
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 2614,
                "end": 2619
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 2633,
                        "end": 2637
                      }
                    },
                    "loc": {
                      "start": 2633,
                      "end": 2637
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 2651,
                            "end": 2659
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2648,
                          "end": 2659
                        }
                      }
                    ],
                    "loc": {
                      "start": 2638,
                      "end": 2665
                    }
                  },
                  "loc": {
                    "start": 2626,
                    "end": 2665
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 2677,
                        "end": 2681
                      }
                    },
                    "loc": {
                      "start": 2677,
                      "end": 2681
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 2695,
                            "end": 2703
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2692,
                          "end": 2703
                        }
                      }
                    ],
                    "loc": {
                      "start": 2682,
                      "end": 2709
                    }
                  },
                  "loc": {
                    "start": 2670,
                    "end": 2709
                  }
                }
              ],
              "loc": {
                "start": 2620,
                "end": 2711
              }
            },
            "loc": {
              "start": 2614,
              "end": 2711
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 2712,
                "end": 2723
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2712,
              "end": 2723
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 2724,
                "end": 2738
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2724,
              "end": 2738
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 2739,
                "end": 2744
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2739,
              "end": 2744
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2745,
                "end": 2754
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2745,
              "end": 2754
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2755,
                "end": 2759
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 2769,
                      "end": 2777
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2766,
                    "end": 2777
                  }
                }
              ],
              "loc": {
                "start": 2760,
                "end": 2779
              }
            },
            "loc": {
              "start": 2755,
              "end": 2779
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 2780,
                "end": 2794
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2780,
              "end": 2794
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 2795,
                "end": 2800
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2795,
              "end": 2800
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2801,
                "end": 2804
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2811,
                      "end": 2820
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2811,
                    "end": 2820
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2825,
                      "end": 2836
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2825,
                    "end": 2836
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 2841,
                      "end": 2852
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2841,
                    "end": 2852
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2857,
                      "end": 2866
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2857,
                    "end": 2866
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2871,
                      "end": 2878
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2871,
                    "end": 2878
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 2883,
                      "end": 2891
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2883,
                    "end": 2891
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2896,
                      "end": 2908
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2896,
                    "end": 2908
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2913,
                      "end": 2921
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2913,
                    "end": 2921
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 2926,
                      "end": 2934
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2926,
                    "end": 2934
                  }
                }
              ],
              "loc": {
                "start": 2805,
                "end": 2936
              }
            },
            "loc": {
              "start": 2801,
              "end": 2936
            }
          }
        ],
        "loc": {
          "start": 2106,
          "end": 2938
        }
      },
      "loc": {
        "start": 2079,
        "end": 2938
      }
    },
    "Note_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_nav",
        "loc": {
          "start": 2948,
          "end": 2956
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 2960,
            "end": 2964
          }
        },
        "loc": {
          "start": 2960,
          "end": 2964
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2967,
                "end": 2969
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2967,
              "end": 2969
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2970,
                "end": 2979
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2970,
              "end": 2979
            }
          }
        ],
        "loc": {
          "start": 2965,
          "end": 2981
        }
      },
      "loc": {
        "start": 2939,
        "end": 2981
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 2991,
          "end": 3003
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3007,
            "end": 3014
          }
        },
        "loc": {
          "start": 3007,
          "end": 3014
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 3017,
                "end": 3025
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 3032,
                      "end": 3044
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 3055,
                            "end": 3057
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3055,
                          "end": 3057
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 3066,
                            "end": 3074
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3066,
                          "end": 3074
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 3083,
                            "end": 3094
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3083,
                          "end": 3094
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 3103,
                            "end": 3107
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3103,
                          "end": 3107
                        }
                      }
                    ],
                    "loc": {
                      "start": 3045,
                      "end": 3113
                    }
                  },
                  "loc": {
                    "start": 3032,
                    "end": 3113
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3118,
                      "end": 3120
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3118,
                    "end": 3120
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3125,
                      "end": 3135
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3125,
                    "end": 3135
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3140,
                      "end": 3150
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3140,
                    "end": 3150
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 3155,
                      "end": 3171
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3155,
                    "end": 3171
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 3176,
                      "end": 3184
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3176,
                    "end": 3184
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 3189,
                      "end": 3198
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3189,
                    "end": 3198
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 3203,
                      "end": 3215
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3203,
                    "end": 3215
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 3220,
                      "end": 3236
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3220,
                    "end": 3236
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 3241,
                      "end": 3251
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3241,
                    "end": 3251
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 3256,
                      "end": 3268
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3256,
                    "end": 3268
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 3273,
                      "end": 3285
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3273,
                    "end": 3285
                  }
                }
              ],
              "loc": {
                "start": 3026,
                "end": 3287
              }
            },
            "loc": {
              "start": 3017,
              "end": 3287
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3288,
                "end": 3290
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3288,
              "end": 3290
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3291,
                "end": 3301
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3291,
              "end": 3301
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3302,
                "end": 3312
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3302,
              "end": 3312
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3313,
                "end": 3322
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3313,
              "end": 3322
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 3323,
                "end": 3334
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3323,
              "end": 3334
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 3335,
                "end": 3341
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 3351,
                      "end": 3361
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3348,
                    "end": 3361
                  }
                }
              ],
              "loc": {
                "start": 3342,
                "end": 3363
              }
            },
            "loc": {
              "start": 3335,
              "end": 3363
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 3364,
                "end": 3369
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 3383,
                        "end": 3387
                      }
                    },
                    "loc": {
                      "start": 3383,
                      "end": 3387
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 3401,
                            "end": 3409
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3398,
                          "end": 3409
                        }
                      }
                    ],
                    "loc": {
                      "start": 3388,
                      "end": 3415
                    }
                  },
                  "loc": {
                    "start": 3376,
                    "end": 3415
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 3427,
                        "end": 3431
                      }
                    },
                    "loc": {
                      "start": 3427,
                      "end": 3431
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 3445,
                            "end": 3453
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3442,
                          "end": 3453
                        }
                      }
                    ],
                    "loc": {
                      "start": 3432,
                      "end": 3459
                    }
                  },
                  "loc": {
                    "start": 3420,
                    "end": 3459
                  }
                }
              ],
              "loc": {
                "start": 3370,
                "end": 3461
              }
            },
            "loc": {
              "start": 3364,
              "end": 3461
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 3462,
                "end": 3473
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3462,
              "end": 3473
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 3474,
                "end": 3488
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3474,
              "end": 3488
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 3489,
                "end": 3494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3489,
              "end": 3494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 3495,
                "end": 3504
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3495,
              "end": 3504
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 3505,
                "end": 3509
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 3519,
                      "end": 3527
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3516,
                    "end": 3527
                  }
                }
              ],
              "loc": {
                "start": 3510,
                "end": 3529
              }
            },
            "loc": {
              "start": 3505,
              "end": 3529
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 3530,
                "end": 3544
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3530,
              "end": 3544
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 3545,
                "end": 3550
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3545,
              "end": 3550
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 3551,
                "end": 3554
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 3561,
                      "end": 3570
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3561,
                    "end": 3570
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 3575,
                      "end": 3586
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3575,
                    "end": 3586
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 3591,
                      "end": 3602
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3591,
                    "end": 3602
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 3607,
                      "end": 3616
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3607,
                    "end": 3616
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 3621,
                      "end": 3628
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3621,
                    "end": 3628
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 3633,
                      "end": 3641
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3633,
                    "end": 3641
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 3646,
                      "end": 3658
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3646,
                    "end": 3658
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 3663,
                      "end": 3671
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3663,
                    "end": 3671
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 3676,
                      "end": 3684
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3676,
                    "end": 3684
                  }
                }
              ],
              "loc": {
                "start": 3555,
                "end": 3686
              }
            },
            "loc": {
              "start": 3551,
              "end": 3686
            }
          }
        ],
        "loc": {
          "start": 3015,
          "end": 3688
        }
      },
      "loc": {
        "start": 2982,
        "end": 3688
      }
    },
    "Project_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_nav",
        "loc": {
          "start": 3698,
          "end": 3709
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3713,
            "end": 3720
          }
        },
        "loc": {
          "start": 3713,
          "end": 3720
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3723,
                "end": 3725
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3723,
              "end": 3725
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3726,
                "end": 3735
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3726,
              "end": 3735
            }
          }
        ],
        "loc": {
          "start": 3721,
          "end": 3737
        }
      },
      "loc": {
        "start": 3689,
        "end": 3737
      }
    },
    "Question_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Question_list",
        "loc": {
          "start": 3747,
          "end": 3760
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Question",
          "loc": {
            "start": 3764,
            "end": 3772
          }
        },
        "loc": {
          "start": 3764,
          "end": 3772
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 3775,
                "end": 3787
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3794,
                      "end": 3796
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3794,
                    "end": 3796
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3801,
                      "end": 3809
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3801,
                    "end": 3809
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3814,
                      "end": 3825
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3814,
                    "end": 3825
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3830,
                      "end": 3834
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3830,
                    "end": 3834
                  }
                }
              ],
              "loc": {
                "start": 3788,
                "end": 3836
              }
            },
            "loc": {
              "start": 3775,
              "end": 3836
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3837,
                "end": 3839
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3837,
              "end": 3839
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3840,
                "end": 3850
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3840,
              "end": 3850
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3851,
                "end": 3861
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3851,
              "end": 3861
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "createdBy",
              "loc": {
                "start": 3862,
                "end": 3871
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3878,
                      "end": 3880
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3878,
                    "end": 3880
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3885,
                      "end": 3895
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3885,
                    "end": 3895
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3900,
                      "end": 3910
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3900,
                    "end": 3910
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
                    "loc": {
                      "start": 3915,
                      "end": 3926
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3915,
                    "end": 3926
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 3931,
                      "end": 3937
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3931,
                    "end": 3937
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBot",
                    "loc": {
                      "start": 3942,
                      "end": 3947
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3942,
                    "end": 3947
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBotDepictingPerson",
                    "loc": {
                      "start": 3952,
                      "end": 3972
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3952,
                    "end": 3972
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3977,
                      "end": 3981
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3977,
                    "end": 3981
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 3986,
                      "end": 3998
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3986,
                    "end": 3998
                  }
                }
              ],
              "loc": {
                "start": 3872,
                "end": 4000
              }
            },
            "loc": {
              "start": 3862,
              "end": 4000
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "hasAcceptedAnswer",
              "loc": {
                "start": 4001,
                "end": 4018
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4001,
              "end": 4018
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4019,
                "end": 4028
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4019,
              "end": 4028
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 4029,
                "end": 4034
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4029,
              "end": 4034
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 4035,
                "end": 4044
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4035,
              "end": 4044
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "answersCount",
              "loc": {
                "start": 4045,
                "end": 4057
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4045,
              "end": 4057
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 4058,
                "end": 4071
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4058,
              "end": 4071
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 4072,
                "end": 4084
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4072,
              "end": 4084
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forObject",
              "loc": {
                "start": 4085,
                "end": 4094
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Api",
                      "loc": {
                        "start": 4108,
                        "end": 4111
                      }
                    },
                    "loc": {
                      "start": 4108,
                      "end": 4111
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Api_nav",
                          "loc": {
                            "start": 4125,
                            "end": 4132
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4122,
                          "end": 4132
                        }
                      }
                    ],
                    "loc": {
                      "start": 4112,
                      "end": 4138
                    }
                  },
                  "loc": {
                    "start": 4101,
                    "end": 4138
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Code",
                      "loc": {
                        "start": 4150,
                        "end": 4154
                      }
                    },
                    "loc": {
                      "start": 4150,
                      "end": 4154
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Code_nav",
                          "loc": {
                            "start": 4168,
                            "end": 4176
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4165,
                          "end": 4176
                        }
                      }
                    ],
                    "loc": {
                      "start": 4155,
                      "end": 4182
                    }
                  },
                  "loc": {
                    "start": 4143,
                    "end": 4182
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Note",
                      "loc": {
                        "start": 4194,
                        "end": 4198
                      }
                    },
                    "loc": {
                      "start": 4194,
                      "end": 4198
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Note_nav",
                          "loc": {
                            "start": 4212,
                            "end": 4220
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4209,
                          "end": 4220
                        }
                      }
                    ],
                    "loc": {
                      "start": 4199,
                      "end": 4226
                    }
                  },
                  "loc": {
                    "start": 4187,
                    "end": 4226
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Project",
                      "loc": {
                        "start": 4238,
                        "end": 4245
                      }
                    },
                    "loc": {
                      "start": 4238,
                      "end": 4245
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Project_nav",
                          "loc": {
                            "start": 4259,
                            "end": 4270
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4256,
                          "end": 4270
                        }
                      }
                    ],
                    "loc": {
                      "start": 4246,
                      "end": 4276
                    }
                  },
                  "loc": {
                    "start": 4231,
                    "end": 4276
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Routine",
                      "loc": {
                        "start": 4288,
                        "end": 4295
                      }
                    },
                    "loc": {
                      "start": 4288,
                      "end": 4295
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Routine_nav",
                          "loc": {
                            "start": 4309,
                            "end": 4320
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4306,
                          "end": 4320
                        }
                      }
                    ],
                    "loc": {
                      "start": 4296,
                      "end": 4326
                    }
                  },
                  "loc": {
                    "start": 4281,
                    "end": 4326
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Standard",
                      "loc": {
                        "start": 4338,
                        "end": 4346
                      }
                    },
                    "loc": {
                      "start": 4338,
                      "end": 4346
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Standard_nav",
                          "loc": {
                            "start": 4360,
                            "end": 4372
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4357,
                          "end": 4372
                        }
                      }
                    ],
                    "loc": {
                      "start": 4347,
                      "end": 4378
                    }
                  },
                  "loc": {
                    "start": 4331,
                    "end": 4378
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 4390,
                        "end": 4394
                      }
                    },
                    "loc": {
                      "start": 4390,
                      "end": 4394
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 4408,
                            "end": 4416
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4405,
                          "end": 4416
                        }
                      }
                    ],
                    "loc": {
                      "start": 4395,
                      "end": 4422
                    }
                  },
                  "loc": {
                    "start": 4383,
                    "end": 4422
                  }
                }
              ],
              "loc": {
                "start": 4095,
                "end": 4424
              }
            },
            "loc": {
              "start": 4085,
              "end": 4424
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 4425,
                "end": 4429
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 4439,
                      "end": 4447
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4436,
                    "end": 4447
                  }
                }
              ],
              "loc": {
                "start": 4430,
                "end": 4449
              }
            },
            "loc": {
              "start": 4425,
              "end": 4449
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 4450,
                "end": 4453
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 4460,
                      "end": 4468
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4460,
                    "end": 4468
                  }
                }
              ],
              "loc": {
                "start": 4454,
                "end": 4470
              }
            },
            "loc": {
              "start": 4450,
              "end": 4470
            }
          }
        ],
        "loc": {
          "start": 3773,
          "end": 4472
        }
      },
      "loc": {
        "start": 3738,
        "end": 4472
      }
    },
    "Routine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_list",
        "loc": {
          "start": 4482,
          "end": 4494
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 4498,
            "end": 4505
          }
        },
        "loc": {
          "start": 4498,
          "end": 4505
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 4508,
                "end": 4516
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 4523,
                      "end": 4535
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4546,
                            "end": 4548
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4546,
                          "end": 4548
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 4557,
                            "end": 4565
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4557,
                          "end": 4565
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 4574,
                            "end": 4585
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4574,
                          "end": 4585
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 4594,
                            "end": 4606
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4594,
                          "end": 4606
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4615,
                            "end": 4619
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4615,
                          "end": 4619
                        }
                      }
                    ],
                    "loc": {
                      "start": 4536,
                      "end": 4625
                    }
                  },
                  "loc": {
                    "start": 4523,
                    "end": 4625
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4630,
                      "end": 4632
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4630,
                    "end": 4632
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4637,
                      "end": 4647
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4637,
                    "end": 4647
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4652,
                      "end": 4662
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4652,
                    "end": 4662
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 4667,
                      "end": 4678
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4667,
                    "end": 4678
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 4683,
                      "end": 4696
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4683,
                    "end": 4696
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 4701,
                      "end": 4711
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4701,
                    "end": 4711
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 4716,
                      "end": 4725
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4716,
                    "end": 4725
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 4730,
                      "end": 4738
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4730,
                    "end": 4738
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 4743,
                      "end": 4752
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4743,
                    "end": 4752
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routineType",
                    "loc": {
                      "start": 4757,
                      "end": 4768
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4757,
                    "end": 4768
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 4773,
                      "end": 4783
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4773,
                    "end": 4783
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 4788,
                      "end": 4800
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4788,
                    "end": 4800
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 4805,
                      "end": 4819
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4805,
                    "end": 4819
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 4824,
                      "end": 4836
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4824,
                    "end": 4836
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 4841,
                      "end": 4853
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4841,
                    "end": 4853
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 4858,
                      "end": 4871
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4858,
                    "end": 4871
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 4876,
                      "end": 4898
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4876,
                    "end": 4898
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 4903,
                      "end": 4913
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4903,
                    "end": 4913
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 4918,
                      "end": 4929
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4918,
                    "end": 4929
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 4934,
                      "end": 4944
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4934,
                    "end": 4944
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 4949,
                      "end": 4963
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4949,
                    "end": 4963
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 4968,
                      "end": 4980
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4968,
                    "end": 4980
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 4985,
                      "end": 4997
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4985,
                    "end": 4997
                  }
                }
              ],
              "loc": {
                "start": 4517,
                "end": 4999
              }
            },
            "loc": {
              "start": 4508,
              "end": 4999
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5000,
                "end": 5002
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5000,
              "end": 5002
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5003,
                "end": 5013
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5003,
              "end": 5013
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5014,
                "end": 5024
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5014,
              "end": 5024
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 5025,
                "end": 5035
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5025,
              "end": 5035
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5036,
                "end": 5045
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5036,
              "end": 5045
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 5046,
                "end": 5057
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5046,
              "end": 5057
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 5058,
                "end": 5064
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 5074,
                      "end": 5084
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5071,
                    "end": 5084
                  }
                }
              ],
              "loc": {
                "start": 5065,
                "end": 5086
              }
            },
            "loc": {
              "start": 5058,
              "end": 5086
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 5087,
                "end": 5092
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 5106,
                        "end": 5110
                      }
                    },
                    "loc": {
                      "start": 5106,
                      "end": 5110
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 5124,
                            "end": 5132
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5121,
                          "end": 5132
                        }
                      }
                    ],
                    "loc": {
                      "start": 5111,
                      "end": 5138
                    }
                  },
                  "loc": {
                    "start": 5099,
                    "end": 5138
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 5150,
                        "end": 5154
                      }
                    },
                    "loc": {
                      "start": 5150,
                      "end": 5154
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 5168,
                            "end": 5176
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5165,
                          "end": 5176
                        }
                      }
                    ],
                    "loc": {
                      "start": 5155,
                      "end": 5182
                    }
                  },
                  "loc": {
                    "start": 5143,
                    "end": 5182
                  }
                }
              ],
              "loc": {
                "start": 5093,
                "end": 5184
              }
            },
            "loc": {
              "start": 5087,
              "end": 5184
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 5185,
                "end": 5196
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5185,
              "end": 5196
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 5197,
                "end": 5211
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5197,
              "end": 5211
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 5212,
                "end": 5217
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5212,
              "end": 5217
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 5218,
                "end": 5227
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5218,
              "end": 5227
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 5228,
                "end": 5232
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 5242,
                      "end": 5250
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5239,
                    "end": 5250
                  }
                }
              ],
              "loc": {
                "start": 5233,
                "end": 5252
              }
            },
            "loc": {
              "start": 5228,
              "end": 5252
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 5253,
                "end": 5267
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5253,
              "end": 5267
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 5268,
                "end": 5273
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5268,
              "end": 5273
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5274,
                "end": 5277
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 5284,
                      "end": 5294
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5284,
                    "end": 5294
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5299,
                      "end": 5308
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5299,
                    "end": 5308
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 5313,
                      "end": 5324
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5313,
                    "end": 5324
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5329,
                      "end": 5338
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5329,
                    "end": 5338
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5343,
                      "end": 5350
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5343,
                    "end": 5350
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 5355,
                      "end": 5363
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5355,
                    "end": 5363
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 5368,
                      "end": 5380
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5368,
                    "end": 5380
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 5385,
                      "end": 5393
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5385,
                    "end": 5393
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 5398,
                      "end": 5406
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5398,
                    "end": 5406
                  }
                }
              ],
              "loc": {
                "start": 5278,
                "end": 5408
              }
            },
            "loc": {
              "start": 5274,
              "end": 5408
            }
          }
        ],
        "loc": {
          "start": 4506,
          "end": 5410
        }
      },
      "loc": {
        "start": 4473,
        "end": 5410
      }
    },
    "Routine_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_nav",
        "loc": {
          "start": 5420,
          "end": 5431
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 5435,
            "end": 5442
          }
        },
        "loc": {
          "start": 5435,
          "end": 5442
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5445,
                "end": 5447
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5445,
              "end": 5447
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 5448,
                "end": 5458
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5448,
              "end": 5458
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5459,
                "end": 5468
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5459,
              "end": 5468
            }
          }
        ],
        "loc": {
          "start": 5443,
          "end": 5470
        }
      },
      "loc": {
        "start": 5411,
        "end": 5470
      }
    },
    "Standard_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_list",
        "loc": {
          "start": 5480,
          "end": 5493
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 5497,
            "end": 5505
          }
        },
        "loc": {
          "start": 5497,
          "end": 5505
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 5508,
                "end": 5516
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 5523,
                      "end": 5535
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5546,
                            "end": 5548
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5546,
                          "end": 5548
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5557,
                            "end": 5565
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5557,
                          "end": 5565
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5574,
                            "end": 5585
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5574,
                          "end": 5585
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 5594,
                            "end": 5606
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5594,
                          "end": 5606
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5615,
                            "end": 5619
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5615,
                          "end": 5619
                        }
                      }
                    ],
                    "loc": {
                      "start": 5536,
                      "end": 5625
                    }
                  },
                  "loc": {
                    "start": 5523,
                    "end": 5625
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5630,
                      "end": 5632
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5630,
                    "end": 5632
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 5637,
                      "end": 5647
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5637,
                    "end": 5647
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 5652,
                      "end": 5662
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5652,
                    "end": 5662
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 5667,
                      "end": 5677
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5667,
                    "end": 5677
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isFile",
                    "loc": {
                      "start": 5682,
                      "end": 5688
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5682,
                    "end": 5688
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5693,
                      "end": 5701
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5693,
                    "end": 5701
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5706,
                      "end": 5715
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5706,
                    "end": 5715
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 5720,
                      "end": 5727
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5720,
                    "end": 5727
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardType",
                    "loc": {
                      "start": 5732,
                      "end": 5744
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5732,
                    "end": 5744
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "props",
                    "loc": {
                      "start": 5749,
                      "end": 5754
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5749,
                    "end": 5754
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yup",
                    "loc": {
                      "start": 5759,
                      "end": 5762
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5759,
                    "end": 5762
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 5767,
                      "end": 5779
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5767,
                    "end": 5779
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 5784,
                      "end": 5796
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5784,
                    "end": 5796
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 5801,
                      "end": 5814
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5801,
                    "end": 5814
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 5819,
                      "end": 5841
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5819,
                    "end": 5841
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 5846,
                      "end": 5856
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5846,
                    "end": 5856
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 5861,
                      "end": 5873
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5861,
                    "end": 5873
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 5878,
                      "end": 5881
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 5892,
                            "end": 5902
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5892,
                          "end": 5902
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 5911,
                            "end": 5918
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5911,
                          "end": 5918
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 5927,
                            "end": 5936
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5927,
                          "end": 5936
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 5945,
                            "end": 5954
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5945,
                          "end": 5954
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 5963,
                            "end": 5972
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5963,
                          "end": 5972
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 5981,
                            "end": 5987
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5981,
                          "end": 5987
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 5996,
                            "end": 6003
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5996,
                          "end": 6003
                        }
                      }
                    ],
                    "loc": {
                      "start": 5882,
                      "end": 6009
                    }
                  },
                  "loc": {
                    "start": 5878,
                    "end": 6009
                  }
                }
              ],
              "loc": {
                "start": 5517,
                "end": 6011
              }
            },
            "loc": {
              "start": 5508,
              "end": 6011
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6012,
                "end": 6014
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6012,
              "end": 6014
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6015,
                "end": 6025
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6015,
              "end": 6025
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6026,
                "end": 6036
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6026,
              "end": 6036
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6037,
                "end": 6046
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6037,
              "end": 6046
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 6047,
                "end": 6058
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6047,
              "end": 6058
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 6059,
                "end": 6065
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 6075,
                      "end": 6085
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6072,
                    "end": 6085
                  }
                }
              ],
              "loc": {
                "start": 6066,
                "end": 6087
              }
            },
            "loc": {
              "start": 6059,
              "end": 6087
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 6088,
                "end": 6093
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 6107,
                        "end": 6111
                      }
                    },
                    "loc": {
                      "start": 6107,
                      "end": 6111
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 6125,
                            "end": 6133
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6122,
                          "end": 6133
                        }
                      }
                    ],
                    "loc": {
                      "start": 6112,
                      "end": 6139
                    }
                  },
                  "loc": {
                    "start": 6100,
                    "end": 6139
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 6151,
                        "end": 6155
                      }
                    },
                    "loc": {
                      "start": 6151,
                      "end": 6155
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 6169,
                            "end": 6177
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6166,
                          "end": 6177
                        }
                      }
                    ],
                    "loc": {
                      "start": 6156,
                      "end": 6183
                    }
                  },
                  "loc": {
                    "start": 6144,
                    "end": 6183
                  }
                }
              ],
              "loc": {
                "start": 6094,
                "end": 6185
              }
            },
            "loc": {
              "start": 6088,
              "end": 6185
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 6186,
                "end": 6197
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6186,
              "end": 6197
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 6198,
                "end": 6212
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6198,
              "end": 6212
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 6213,
                "end": 6218
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6213,
              "end": 6218
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6219,
                "end": 6228
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6219,
              "end": 6228
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 6229,
                "end": 6233
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 6243,
                      "end": 6251
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6240,
                    "end": 6251
                  }
                }
              ],
              "loc": {
                "start": 6234,
                "end": 6253
              }
            },
            "loc": {
              "start": 6229,
              "end": 6253
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 6254,
                "end": 6268
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6254,
              "end": 6268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 6269,
                "end": 6274
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6269,
              "end": 6274
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6275,
                "end": 6278
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6285,
                      "end": 6294
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6285,
                    "end": 6294
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 6299,
                      "end": 6310
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6299,
                    "end": 6310
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 6315,
                      "end": 6326
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6315,
                    "end": 6326
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6331,
                      "end": 6340
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6331,
                    "end": 6340
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6345,
                      "end": 6352
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6345,
                    "end": 6352
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 6357,
                      "end": 6365
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6357,
                    "end": 6365
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6370,
                      "end": 6382
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6370,
                    "end": 6382
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 6387,
                      "end": 6395
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6387,
                    "end": 6395
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 6400,
                      "end": 6408
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6400,
                    "end": 6408
                  }
                }
              ],
              "loc": {
                "start": 6279,
                "end": 6410
              }
            },
            "loc": {
              "start": 6275,
              "end": 6410
            }
          }
        ],
        "loc": {
          "start": 5506,
          "end": 6412
        }
      },
      "loc": {
        "start": 5471,
        "end": 6412
      }
    },
    "Standard_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_nav",
        "loc": {
          "start": 6422,
          "end": 6434
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 6438,
            "end": 6446
          }
        },
        "loc": {
          "start": 6438,
          "end": 6446
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6449,
                "end": 6451
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6449,
              "end": 6451
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6452,
                "end": 6461
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6452,
              "end": 6461
            }
          }
        ],
        "loc": {
          "start": 6447,
          "end": 6463
        }
      },
      "loc": {
        "start": 6413,
        "end": 6463
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 6473,
          "end": 6481
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 6485,
            "end": 6488
          }
        },
        "loc": {
          "start": 6485,
          "end": 6488
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6491,
                "end": 6493
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6491,
              "end": 6493
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6494,
                "end": 6504
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6494,
              "end": 6504
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 6505,
                "end": 6508
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6505,
              "end": 6508
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6509,
                "end": 6518
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6509,
              "end": 6518
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 6519,
                "end": 6531
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6538,
                      "end": 6540
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6538,
                    "end": 6540
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6545,
                      "end": 6553
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6545,
                    "end": 6553
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 6558,
                      "end": 6569
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6558,
                    "end": 6569
                  }
                }
              ],
              "loc": {
                "start": 6532,
                "end": 6571
              }
            },
            "loc": {
              "start": 6519,
              "end": 6571
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6572,
                "end": 6575
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isOwn",
                    "loc": {
                      "start": 6582,
                      "end": 6587
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6582,
                    "end": 6587
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6592,
                      "end": 6604
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6592,
                    "end": 6604
                  }
                }
              ],
              "loc": {
                "start": 6576,
                "end": 6606
              }
            },
            "loc": {
              "start": 6572,
              "end": 6606
            }
          }
        ],
        "loc": {
          "start": 6489,
          "end": 6608
        }
      },
      "loc": {
        "start": 6464,
        "end": 6608
      }
    },
    "Team_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_list",
        "loc": {
          "start": 6618,
          "end": 6627
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 6631,
            "end": 6635
          }
        },
        "loc": {
          "start": 6631,
          "end": 6635
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6638,
                "end": 6640
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6638,
              "end": 6640
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 6641,
                "end": 6652
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6641,
              "end": 6652
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 6653,
                "end": 6659
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6653,
              "end": 6659
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6660,
                "end": 6670
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6660,
              "end": 6670
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6671,
                "end": 6681
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6671,
              "end": 6681
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOpenToNewMembers",
              "loc": {
                "start": 6682,
                "end": 6700
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6682,
              "end": 6700
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6701,
                "end": 6710
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6701,
              "end": 6710
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 6711,
                "end": 6724
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6711,
              "end": 6724
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membersCount",
              "loc": {
                "start": 6725,
                "end": 6737
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6725,
              "end": 6737
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 6738,
                "end": 6750
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6738,
              "end": 6750
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 6751,
                "end": 6763
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6751,
              "end": 6763
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6764,
                "end": 6773
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6764,
              "end": 6773
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 6774,
                "end": 6778
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 6788,
                      "end": 6796
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6785,
                    "end": 6796
                  }
                }
              ],
              "loc": {
                "start": 6779,
                "end": 6798
              }
            },
            "loc": {
              "start": 6774,
              "end": 6798
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 6799,
                "end": 6811
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6818,
                      "end": 6820
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6818,
                    "end": 6820
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6825,
                      "end": 6833
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6825,
                    "end": 6833
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 6838,
                      "end": 6841
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6838,
                    "end": 6841
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6846,
                      "end": 6850
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6846,
                    "end": 6850
                  }
                }
              ],
              "loc": {
                "start": 6812,
                "end": 6852
              }
            },
            "loc": {
              "start": 6799,
              "end": 6852
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6853,
                "end": 6856
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 6863,
                      "end": 6876
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6863,
                    "end": 6876
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6881,
                      "end": 6890
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6881,
                    "end": 6890
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 6895,
                      "end": 6906
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6895,
                    "end": 6906
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 6911,
                      "end": 6920
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6911,
                    "end": 6920
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6925,
                      "end": 6934
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6925,
                    "end": 6934
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6939,
                      "end": 6946
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6939,
                    "end": 6946
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6951,
                      "end": 6963
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6951,
                    "end": 6963
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 6968,
                      "end": 6976
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6968,
                    "end": 6976
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 6981,
                      "end": 6995
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 7006,
                            "end": 7008
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7006,
                          "end": 7008
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 7017,
                            "end": 7027
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7017,
                          "end": 7027
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 7036,
                            "end": 7046
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7036,
                          "end": 7046
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 7055,
                            "end": 7062
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7055,
                          "end": 7062
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 7071,
                            "end": 7082
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7071,
                          "end": 7082
                        }
                      }
                    ],
                    "loc": {
                      "start": 6996,
                      "end": 7088
                    }
                  },
                  "loc": {
                    "start": 6981,
                    "end": 7088
                  }
                }
              ],
              "loc": {
                "start": 6857,
                "end": 7090
              }
            },
            "loc": {
              "start": 6853,
              "end": 7090
            }
          }
        ],
        "loc": {
          "start": 6636,
          "end": 7092
        }
      },
      "loc": {
        "start": 6609,
        "end": 7092
      }
    },
    "Team_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_nav",
        "loc": {
          "start": 7102,
          "end": 7110
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 7114,
            "end": 7118
          }
        },
        "loc": {
          "start": 7114,
          "end": 7118
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7121,
                "end": 7123
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7121,
              "end": 7123
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7124,
                "end": 7135
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7124,
              "end": 7135
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7136,
                "end": 7142
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7136,
              "end": 7142
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7143,
                "end": 7155
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7143,
              "end": 7155
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7156,
                "end": 7159
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 7166,
                      "end": 7179
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7166,
                    "end": 7179
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 7184,
                      "end": 7193
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7184,
                    "end": 7193
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 7198,
                      "end": 7209
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7198,
                    "end": 7209
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 7214,
                      "end": 7223
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7214,
                    "end": 7223
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7228,
                      "end": 7237
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7228,
                    "end": 7237
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7242,
                      "end": 7249
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7242,
                    "end": 7249
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7254,
                      "end": 7266
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7254,
                    "end": 7266
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7271,
                      "end": 7279
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7271,
                    "end": 7279
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 7284,
                      "end": 7298
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 7309,
                            "end": 7311
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7309,
                          "end": 7311
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 7320,
                            "end": 7330
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7320,
                          "end": 7330
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 7339,
                            "end": 7349
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7339,
                          "end": 7349
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 7358,
                            "end": 7365
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7358,
                          "end": 7365
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 7374,
                            "end": 7385
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7374,
                          "end": 7385
                        }
                      }
                    ],
                    "loc": {
                      "start": 7299,
                      "end": 7391
                    }
                  },
                  "loc": {
                    "start": 7284,
                    "end": 7391
                  }
                }
              ],
              "loc": {
                "start": 7160,
                "end": 7393
              }
            },
            "loc": {
              "start": 7156,
              "end": 7393
            }
          }
        ],
        "loc": {
          "start": 7119,
          "end": 7395
        }
      },
      "loc": {
        "start": 7093,
        "end": 7395
      }
    },
    "User_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_list",
        "loc": {
          "start": 7405,
          "end": 7414
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7418,
            "end": 7422
          }
        },
        "loc": {
          "start": 7418,
          "end": 7422
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 7425,
                "end": 7437
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7444,
                      "end": 7446
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7444,
                    "end": 7446
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 7451,
                      "end": 7459
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7451,
                    "end": 7459
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 7464,
                      "end": 7467
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7464,
                    "end": 7467
                  }
                }
              ],
              "loc": {
                "start": 7438,
                "end": 7469
              }
            },
            "loc": {
              "start": 7425,
              "end": 7469
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7470,
                "end": 7472
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7470,
              "end": 7472
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7473,
                "end": 7483
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7473,
              "end": 7483
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7484,
                "end": 7494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7484,
              "end": 7494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7495,
                "end": 7506
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7495,
              "end": 7506
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7507,
                "end": 7513
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7507,
              "end": 7513
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7514,
                "end": 7519
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7514,
              "end": 7519
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 7520,
                "end": 7540
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7520,
              "end": 7540
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7541,
                "end": 7545
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7541,
              "end": 7545
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7546,
                "end": 7558
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7546,
              "end": 7558
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7559,
                "end": 7568
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7559,
              "end": 7568
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsReceivedCount",
              "loc": {
                "start": 7569,
                "end": 7589
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7569,
              "end": 7589
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7590,
                "end": 7593
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 7600,
                      "end": 7609
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7600,
                    "end": 7609
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 7614,
                      "end": 7623
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7614,
                    "end": 7623
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7628,
                      "end": 7637
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7628,
                    "end": 7637
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7642,
                      "end": 7654
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7642,
                    "end": 7654
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7659,
                      "end": 7667
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7659,
                    "end": 7667
                  }
                }
              ],
              "loc": {
                "start": 7594,
                "end": 7669
              }
            },
            "loc": {
              "start": 7590,
              "end": 7669
            }
          }
        ],
        "loc": {
          "start": 7423,
          "end": 7671
        }
      },
      "loc": {
        "start": 7396,
        "end": 7671
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 7681,
          "end": 7689
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7693,
            "end": 7697
          }
        },
        "loc": {
          "start": 7693,
          "end": 7697
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7700,
                "end": 7702
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7700,
              "end": 7702
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7703,
                "end": 7713
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7703,
              "end": 7713
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7714,
                "end": 7724
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7714,
              "end": 7724
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7725,
                "end": 7736
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7725,
              "end": 7736
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7737,
                "end": 7743
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7737,
              "end": 7743
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7744,
                "end": 7749
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7744,
              "end": 7749
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 7750,
                "end": 7770
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7750,
              "end": 7770
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7771,
                "end": 7775
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7771,
              "end": 7775
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7776,
                "end": 7788
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7776,
              "end": 7788
            }
          }
        ],
        "loc": {
          "start": 7698,
          "end": 7790
        }
      },
      "loc": {
        "start": 7672,
        "end": 7790
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "populars",
      "loc": {
        "start": 7798,
        "end": 7806
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 7808,
              "end": 7813
            }
          },
          "loc": {
            "start": 7807,
            "end": 7813
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "PopularSearchInput",
              "loc": {
                "start": 7815,
                "end": 7833
              }
            },
            "loc": {
              "start": 7815,
              "end": 7833
            }
          },
          "loc": {
            "start": 7815,
            "end": 7834
          }
        },
        "directives": [],
        "loc": {
          "start": 7807,
          "end": 7834
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "populars",
            "loc": {
              "start": 7840,
              "end": 7848
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 7849,
                  "end": 7854
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 7857,
                    "end": 7862
                  }
                },
                "loc": {
                  "start": 7856,
                  "end": 7862
                }
              },
              "loc": {
                "start": 7849,
                "end": 7862
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "edges",
                  "loc": {
                    "start": 7870,
                    "end": 7875
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "cursor",
                        "loc": {
                          "start": 7886,
                          "end": 7892
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 7886,
                        "end": 7892
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 7901,
                          "end": 7905
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Api",
                                "loc": {
                                  "start": 7927,
                                  "end": 7930
                                }
                              },
                              "loc": {
                                "start": 7927,
                                "end": 7930
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Api_list",
                                    "loc": {
                                      "start": 7952,
                                      "end": 7960
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 7949,
                                    "end": 7960
                                  }
                                }
                              ],
                              "loc": {
                                "start": 7931,
                                "end": 7974
                              }
                            },
                            "loc": {
                              "start": 7920,
                              "end": 7974
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Code",
                                "loc": {
                                  "start": 7994,
                                  "end": 7998
                                }
                              },
                              "loc": {
                                "start": 7994,
                                "end": 7998
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Code_list",
                                    "loc": {
                                      "start": 8020,
                                      "end": 8029
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8017,
                                    "end": 8029
                                  }
                                }
                              ],
                              "loc": {
                                "start": 7999,
                                "end": 8043
                              }
                            },
                            "loc": {
                              "start": 7987,
                              "end": 8043
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Note",
                                "loc": {
                                  "start": 8063,
                                  "end": 8067
                                }
                              },
                              "loc": {
                                "start": 8063,
                                "end": 8067
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Note_list",
                                    "loc": {
                                      "start": 8089,
                                      "end": 8098
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8086,
                                    "end": 8098
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8068,
                                "end": 8112
                              }
                            },
                            "loc": {
                              "start": 8056,
                              "end": 8112
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Project",
                                "loc": {
                                  "start": 8132,
                                  "end": 8139
                                }
                              },
                              "loc": {
                                "start": 8132,
                                "end": 8139
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Project_list",
                                    "loc": {
                                      "start": 8161,
                                      "end": 8173
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8158,
                                    "end": 8173
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8140,
                                "end": 8187
                              }
                            },
                            "loc": {
                              "start": 8125,
                              "end": 8187
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Question",
                                "loc": {
                                  "start": 8207,
                                  "end": 8215
                                }
                              },
                              "loc": {
                                "start": 8207,
                                "end": 8215
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Question_list",
                                    "loc": {
                                      "start": 8237,
                                      "end": 8250
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8234,
                                    "end": 8250
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8216,
                                "end": 8264
                              }
                            },
                            "loc": {
                              "start": 8200,
                              "end": 8264
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Routine",
                                "loc": {
                                  "start": 8284,
                                  "end": 8291
                                }
                              },
                              "loc": {
                                "start": 8284,
                                "end": 8291
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Routine_list",
                                    "loc": {
                                      "start": 8313,
                                      "end": 8325
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8310,
                                    "end": 8325
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8292,
                                "end": 8339
                              }
                            },
                            "loc": {
                              "start": 8277,
                              "end": 8339
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Standard",
                                "loc": {
                                  "start": 8359,
                                  "end": 8367
                                }
                              },
                              "loc": {
                                "start": 8359,
                                "end": 8367
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Standard_list",
                                    "loc": {
                                      "start": 8389,
                                      "end": 8402
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8386,
                                    "end": 8402
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8368,
                                "end": 8416
                              }
                            },
                            "loc": {
                              "start": 8352,
                              "end": 8416
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Team",
                                "loc": {
                                  "start": 8436,
                                  "end": 8440
                                }
                              },
                              "loc": {
                                "start": 8436,
                                "end": 8440
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Team_list",
                                    "loc": {
                                      "start": 8462,
                                      "end": 8471
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8459,
                                    "end": 8471
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8441,
                                "end": 8485
                              }
                            },
                            "loc": {
                              "start": 8429,
                              "end": 8485
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "User",
                                "loc": {
                                  "start": 8505,
                                  "end": 8509
                                }
                              },
                              "loc": {
                                "start": 8505,
                                "end": 8509
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "User_list",
                                    "loc": {
                                      "start": 8531,
                                      "end": 8540
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8528,
                                    "end": 8540
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8510,
                                "end": 8554
                              }
                            },
                            "loc": {
                              "start": 8498,
                              "end": 8554
                            }
                          }
                        ],
                        "loc": {
                          "start": 7906,
                          "end": 8564
                        }
                      },
                      "loc": {
                        "start": 7901,
                        "end": 8564
                      }
                    }
                  ],
                  "loc": {
                    "start": 7876,
                    "end": 8570
                  }
                },
                "loc": {
                  "start": 7870,
                  "end": 8570
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 8575,
                    "end": 8583
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasNextPage",
                        "loc": {
                          "start": 8594,
                          "end": 8605
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8594,
                        "end": 8605
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorApi",
                        "loc": {
                          "start": 8614,
                          "end": 8626
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8614,
                        "end": 8626
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorCode",
                        "loc": {
                          "start": 8635,
                          "end": 8648
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8635,
                        "end": 8648
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorNote",
                        "loc": {
                          "start": 8657,
                          "end": 8670
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8657,
                        "end": 8670
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 8679,
                          "end": 8695
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8679,
                        "end": 8695
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorQuestion",
                        "loc": {
                          "start": 8704,
                          "end": 8721
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8704,
                        "end": 8721
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRoutine",
                        "loc": {
                          "start": 8730,
                          "end": 8746
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8730,
                        "end": 8746
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorStandard",
                        "loc": {
                          "start": 8755,
                          "end": 8772
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8755,
                        "end": 8772
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorTeam",
                        "loc": {
                          "start": 8781,
                          "end": 8794
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8781,
                        "end": 8794
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorUser",
                        "loc": {
                          "start": 8803,
                          "end": 8816
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8803,
                        "end": 8816
                      }
                    }
                  ],
                  "loc": {
                    "start": 8584,
                    "end": 8822
                  }
                },
                "loc": {
                  "start": 8575,
                  "end": 8822
                }
              }
            ],
            "loc": {
              "start": 7864,
              "end": 8826
            }
          },
          "loc": {
            "start": 7840,
            "end": 8826
          }
        }
      ],
      "loc": {
        "start": 7836,
        "end": 8828
      }
    },
    "loc": {
      "start": 7792,
      "end": 8828
    }
  },
  "variableValues": {},
  "path": {
    "key": "popular_findMany"
  }
} as const;
