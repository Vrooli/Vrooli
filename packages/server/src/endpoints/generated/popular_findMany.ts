export const popular_findMany = {
  "fieldName": "populars",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "populars",
        "loc": {
          "start": 7856,
          "end": 7864
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 7865,
              "end": 7870
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 7873,
                "end": 7878
              }
            },
            "loc": {
              "start": 7872,
              "end": 7878
            }
          },
          "loc": {
            "start": 7865,
            "end": 7878
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
                "start": 7886,
                "end": 7891
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
                      "start": 7902,
                      "end": 7908
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7902,
                    "end": 7908
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 7917,
                      "end": 7921
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
                              "start": 7943,
                              "end": 7946
                            }
                          },
                          "loc": {
                            "start": 7943,
                            "end": 7946
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
                                  "start": 7968,
                                  "end": 7976
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 7965,
                                "end": 7976
                              }
                            }
                          ],
                          "loc": {
                            "start": 7947,
                            "end": 7990
                          }
                        },
                        "loc": {
                          "start": 7936,
                          "end": 7990
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
                              "start": 8010,
                              "end": 8014
                            }
                          },
                          "loc": {
                            "start": 8010,
                            "end": 8014
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
                                  "start": 8036,
                                  "end": 8045
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8033,
                                "end": 8045
                              }
                            }
                          ],
                          "loc": {
                            "start": 8015,
                            "end": 8059
                          }
                        },
                        "loc": {
                          "start": 8003,
                          "end": 8059
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
                              "start": 8079,
                              "end": 8083
                            }
                          },
                          "loc": {
                            "start": 8079,
                            "end": 8083
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
                                  "start": 8105,
                                  "end": 8114
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8102,
                                "end": 8114
                              }
                            }
                          ],
                          "loc": {
                            "start": 8084,
                            "end": 8128
                          }
                        },
                        "loc": {
                          "start": 8072,
                          "end": 8128
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
                              "start": 8148,
                              "end": 8155
                            }
                          },
                          "loc": {
                            "start": 8148,
                            "end": 8155
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
                                  "start": 8177,
                                  "end": 8189
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8174,
                                "end": 8189
                              }
                            }
                          ],
                          "loc": {
                            "start": 8156,
                            "end": 8203
                          }
                        },
                        "loc": {
                          "start": 8141,
                          "end": 8203
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
                              "start": 8223,
                              "end": 8231
                            }
                          },
                          "loc": {
                            "start": 8223,
                            "end": 8231
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
                                  "start": 8253,
                                  "end": 8266
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8250,
                                "end": 8266
                              }
                            }
                          ],
                          "loc": {
                            "start": 8232,
                            "end": 8280
                          }
                        },
                        "loc": {
                          "start": 8216,
                          "end": 8280
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
                              "start": 8300,
                              "end": 8307
                            }
                          },
                          "loc": {
                            "start": 8300,
                            "end": 8307
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
                                  "start": 8329,
                                  "end": 8341
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8326,
                                "end": 8341
                              }
                            }
                          ],
                          "loc": {
                            "start": 8308,
                            "end": 8355
                          }
                        },
                        "loc": {
                          "start": 8293,
                          "end": 8355
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
                              "start": 8375,
                              "end": 8383
                            }
                          },
                          "loc": {
                            "start": 8375,
                            "end": 8383
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
                                  "start": 8405,
                                  "end": 8418
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8402,
                                "end": 8418
                              }
                            }
                          ],
                          "loc": {
                            "start": 8384,
                            "end": 8432
                          }
                        },
                        "loc": {
                          "start": 8368,
                          "end": 8432
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
                              "start": 8452,
                              "end": 8456
                            }
                          },
                          "loc": {
                            "start": 8452,
                            "end": 8456
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
                                  "start": 8478,
                                  "end": 8487
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8475,
                                "end": 8487
                              }
                            }
                          ],
                          "loc": {
                            "start": 8457,
                            "end": 8501
                          }
                        },
                        "loc": {
                          "start": 8445,
                          "end": 8501
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
                              "start": 8521,
                              "end": 8525
                            }
                          },
                          "loc": {
                            "start": 8521,
                            "end": 8525
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
                                  "start": 8547,
                                  "end": 8556
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8544,
                                "end": 8556
                              }
                            }
                          ],
                          "loc": {
                            "start": 8526,
                            "end": 8570
                          }
                        },
                        "loc": {
                          "start": 8514,
                          "end": 8570
                        }
                      }
                    ],
                    "loc": {
                      "start": 7922,
                      "end": 8580
                    }
                  },
                  "loc": {
                    "start": 7917,
                    "end": 8580
                  }
                }
              ],
              "loc": {
                "start": 7892,
                "end": 8586
              }
            },
            "loc": {
              "start": 7886,
              "end": 8586
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 8591,
                "end": 8599
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
                      "start": 8610,
                      "end": 8621
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8610,
                    "end": 8621
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorApi",
                    "loc": {
                      "start": 8630,
                      "end": 8642
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8630,
                    "end": 8642
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorCode",
                    "loc": {
                      "start": 8651,
                      "end": 8664
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8651,
                    "end": 8664
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorNote",
                    "loc": {
                      "start": 8673,
                      "end": 8686
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8673,
                    "end": 8686
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 8695,
                      "end": 8711
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8695,
                    "end": 8711
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorQuestion",
                    "loc": {
                      "start": 8720,
                      "end": 8737
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8720,
                    "end": 8737
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRoutine",
                    "loc": {
                      "start": 8746,
                      "end": 8762
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8746,
                    "end": 8762
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorStandard",
                    "loc": {
                      "start": 8771,
                      "end": 8788
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8771,
                    "end": 8788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorTeam",
                    "loc": {
                      "start": 8797,
                      "end": 8810
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8797,
                    "end": 8810
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorUser",
                    "loc": {
                      "start": 8819,
                      "end": 8832
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8819,
                    "end": 8832
                  }
                }
              ],
              "loc": {
                "start": 8600,
                "end": 8838
              }
            },
            "loc": {
              "start": 8591,
              "end": 8838
            }
          }
        ],
        "loc": {
          "start": 7880,
          "end": 8842
        }
      },
      "loc": {
        "start": 7856,
        "end": 8842
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
              "value": "default",
              "loc": {
                "start": 1144,
                "end": 1151
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1144,
              "end": 1151
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contractType",
              "loc": {
                "start": 1156,
                "end": 1168
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1156,
              "end": 1168
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "content",
              "loc": {
                "start": 1173,
                "end": 1180
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1173,
              "end": 1180
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1185,
                "end": 1197
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1185,
              "end": 1197
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1202,
                "end": 1214
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1202,
              "end": 1214
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 1219,
                "end": 1232
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1219,
              "end": 1232
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 1237,
                "end": 1259
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1237,
              "end": 1259
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 1264,
                "end": 1274
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1264,
              "end": 1274
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1279,
                "end": 1291
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1279,
              "end": 1291
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1296,
                "end": 1299
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
                      "start": 1310,
                      "end": 1320
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1310,
                    "end": 1320
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 1329,
                      "end": 1336
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1329,
                    "end": 1336
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1345,
                      "end": 1354
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1345,
                    "end": 1354
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1363,
                      "end": 1372
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1363,
                    "end": 1372
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1381,
                      "end": 1390
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1381,
                    "end": 1390
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 1399,
                      "end": 1405
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1399,
                    "end": 1405
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1414,
                      "end": 1421
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1414,
                    "end": 1421
                  }
                }
              ],
              "loc": {
                "start": 1300,
                "end": 1427
              }
            },
            "loc": {
              "start": 1296,
              "end": 1427
            }
          }
        ],
        "loc": {
          "start": 938,
          "end": 1429
        }
      },
      "loc": {
        "start": 929,
        "end": 1429
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1430,
          "end": 1432
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1430,
        "end": 1432
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1433,
          "end": 1443
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1433,
        "end": 1443
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1444,
          "end": 1454
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1444,
        "end": 1454
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1455,
          "end": 1464
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1455,
        "end": 1464
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 1465,
          "end": 1476
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1465,
        "end": 1476
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1477,
          "end": 1483
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
                "start": 1493,
                "end": 1503
              }
            },
            "directives": [],
            "loc": {
              "start": 1490,
              "end": 1503
            }
          }
        ],
        "loc": {
          "start": 1484,
          "end": 1505
        }
      },
      "loc": {
        "start": 1477,
        "end": 1505
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1506,
          "end": 1511
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
                  "start": 1525,
                  "end": 1529
                }
              },
              "loc": {
                "start": 1525,
                "end": 1529
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
                      "start": 1543,
                      "end": 1551
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1540,
                    "end": 1551
                  }
                }
              ],
              "loc": {
                "start": 1530,
                "end": 1557
              }
            },
            "loc": {
              "start": 1518,
              "end": 1557
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
                  "start": 1569,
                  "end": 1573
                }
              },
              "loc": {
                "start": 1569,
                "end": 1573
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
                      "start": 1587,
                      "end": 1595
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1584,
                    "end": 1595
                  }
                }
              ],
              "loc": {
                "start": 1574,
                "end": 1601
              }
            },
            "loc": {
              "start": 1562,
              "end": 1601
            }
          }
        ],
        "loc": {
          "start": 1512,
          "end": 1603
        }
      },
      "loc": {
        "start": 1506,
        "end": 1603
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1604,
          "end": 1615
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1604,
        "end": 1615
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 1616,
          "end": 1630
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1616,
        "end": 1630
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 1631,
          "end": 1636
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1631,
        "end": 1636
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1637,
          "end": 1646
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1637,
        "end": 1646
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1647,
          "end": 1651
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
                "start": 1661,
                "end": 1669
              }
            },
            "directives": [],
            "loc": {
              "start": 1658,
              "end": 1669
            }
          }
        ],
        "loc": {
          "start": 1652,
          "end": 1671
        }
      },
      "loc": {
        "start": 1647,
        "end": 1671
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 1672,
          "end": 1686
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1672,
        "end": 1686
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 1687,
          "end": 1692
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1687,
        "end": 1692
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1693,
          "end": 1696
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
                "start": 1703,
                "end": 1712
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1703,
              "end": 1712
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1717,
                "end": 1728
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1717,
              "end": 1728
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 1733,
                "end": 1744
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1733,
              "end": 1744
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1749,
                "end": 1758
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1749,
              "end": 1758
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1763,
                "end": 1770
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1763,
              "end": 1770
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1775,
                "end": 1783
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1775,
              "end": 1783
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1788,
                "end": 1800
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1788,
              "end": 1800
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1805,
                "end": 1813
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1805,
              "end": 1813
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1818,
                "end": 1826
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1818,
              "end": 1826
            }
          }
        ],
        "loc": {
          "start": 1697,
          "end": 1828
        }
      },
      "loc": {
        "start": 1693,
        "end": 1828
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1859,
          "end": 1861
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1859,
        "end": 1861
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1862,
          "end": 1871
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1862,
        "end": 1871
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1905,
          "end": 1907
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1905,
        "end": 1907
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1908,
          "end": 1918
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1908,
        "end": 1918
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1919,
          "end": 1929
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1919,
        "end": 1929
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 1930,
          "end": 1935
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1930,
        "end": 1935
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 1936,
          "end": 1941
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1936,
        "end": 1941
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1942,
          "end": 1947
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
                  "start": 1961,
                  "end": 1965
                }
              },
              "loc": {
                "start": 1961,
                "end": 1965
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
                      "start": 1979,
                      "end": 1987
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1976,
                    "end": 1987
                  }
                }
              ],
              "loc": {
                "start": 1966,
                "end": 1993
              }
            },
            "loc": {
              "start": 1954,
              "end": 1993
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
                  "start": 2005,
                  "end": 2009
                }
              },
              "loc": {
                "start": 2005,
                "end": 2009
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
                      "start": 2023,
                      "end": 2031
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2020,
                    "end": 2031
                  }
                }
              ],
              "loc": {
                "start": 2010,
                "end": 2037
              }
            },
            "loc": {
              "start": 1998,
              "end": 2037
            }
          }
        ],
        "loc": {
          "start": 1948,
          "end": 2039
        }
      },
      "loc": {
        "start": 1942,
        "end": 2039
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2040,
          "end": 2043
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
                "start": 2050,
                "end": 2059
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2050,
              "end": 2059
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2064,
                "end": 2073
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2064,
              "end": 2073
            }
          }
        ],
        "loc": {
          "start": 2044,
          "end": 2075
        }
      },
      "loc": {
        "start": 2040,
        "end": 2075
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 2107,
          "end": 2115
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
                "start": 2122,
                "end": 2134
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
                      "start": 2145,
                      "end": 2147
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2145,
                    "end": 2147
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2156,
                      "end": 2164
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2156,
                    "end": 2164
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2173,
                      "end": 2184
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2173,
                    "end": 2184
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2193,
                      "end": 2197
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2193,
                    "end": 2197
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "pages",
                    "loc": {
                      "start": 2206,
                      "end": 2211
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
                            "start": 2226,
                            "end": 2228
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2226,
                          "end": 2228
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pageIndex",
                          "loc": {
                            "start": 2241,
                            "end": 2250
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2241,
                          "end": 2250
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 2263,
                            "end": 2267
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2263,
                          "end": 2267
                        }
                      }
                    ],
                    "loc": {
                      "start": 2212,
                      "end": 2277
                    }
                  },
                  "loc": {
                    "start": 2206,
                    "end": 2277
                  }
                }
              ],
              "loc": {
                "start": 2135,
                "end": 2283
              }
            },
            "loc": {
              "start": 2122,
              "end": 2283
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2288,
                "end": 2290
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2288,
              "end": 2290
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2295,
                "end": 2305
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2295,
              "end": 2305
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2310,
                "end": 2320
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2310,
              "end": 2320
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 2325,
                "end": 2333
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2325,
              "end": 2333
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2338,
                "end": 2347
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2338,
              "end": 2347
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 2352,
                "end": 2364
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2352,
              "end": 2364
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 2369,
                "end": 2381
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2369,
              "end": 2381
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 2386,
                "end": 2398
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2386,
              "end": 2398
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2403,
                "end": 2406
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
                      "start": 2417,
                      "end": 2427
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2417,
                    "end": 2427
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 2436,
                      "end": 2443
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2436,
                    "end": 2443
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2452,
                      "end": 2461
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2452,
                    "end": 2461
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2470,
                      "end": 2479
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2470,
                    "end": 2479
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2488,
                      "end": 2497
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2488,
                    "end": 2497
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 2506,
                      "end": 2512
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2506,
                    "end": 2512
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2521,
                      "end": 2528
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2521,
                    "end": 2528
                  }
                }
              ],
              "loc": {
                "start": 2407,
                "end": 2534
              }
            },
            "loc": {
              "start": 2403,
              "end": 2534
            }
          }
        ],
        "loc": {
          "start": 2116,
          "end": 2536
        }
      },
      "loc": {
        "start": 2107,
        "end": 2536
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2537,
          "end": 2539
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2537,
        "end": 2539
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2540,
          "end": 2550
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2540,
        "end": 2550
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2551,
          "end": 2561
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2551,
        "end": 2561
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2562,
          "end": 2571
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2562,
        "end": 2571
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 2572,
          "end": 2583
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2572,
        "end": 2583
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 2584,
          "end": 2590
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
                "start": 2600,
                "end": 2610
              }
            },
            "directives": [],
            "loc": {
              "start": 2597,
              "end": 2610
            }
          }
        ],
        "loc": {
          "start": 2591,
          "end": 2612
        }
      },
      "loc": {
        "start": 2584,
        "end": 2612
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 2613,
          "end": 2618
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
                  "start": 2632,
                  "end": 2636
                }
              },
              "loc": {
                "start": 2632,
                "end": 2636
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
                      "start": 2650,
                      "end": 2658
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2647,
                    "end": 2658
                  }
                }
              ],
              "loc": {
                "start": 2637,
                "end": 2664
              }
            },
            "loc": {
              "start": 2625,
              "end": 2664
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
                  "start": 2676,
                  "end": 2680
                }
              },
              "loc": {
                "start": 2676,
                "end": 2680
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
                      "start": 2694,
                      "end": 2702
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2691,
                    "end": 2702
                  }
                }
              ],
              "loc": {
                "start": 2681,
                "end": 2708
              }
            },
            "loc": {
              "start": 2669,
              "end": 2708
            }
          }
        ],
        "loc": {
          "start": 2619,
          "end": 2710
        }
      },
      "loc": {
        "start": 2613,
        "end": 2710
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 2711,
          "end": 2722
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2711,
        "end": 2722
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 2723,
          "end": 2737
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2723,
        "end": 2737
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 2738,
          "end": 2743
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2738,
        "end": 2743
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2744,
          "end": 2753
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2744,
        "end": 2753
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 2754,
          "end": 2758
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
                "start": 2768,
                "end": 2776
              }
            },
            "directives": [],
            "loc": {
              "start": 2765,
              "end": 2776
            }
          }
        ],
        "loc": {
          "start": 2759,
          "end": 2778
        }
      },
      "loc": {
        "start": 2754,
        "end": 2778
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 2779,
          "end": 2793
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2779,
        "end": 2793
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 2794,
          "end": 2799
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2794,
        "end": 2799
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2800,
          "end": 2803
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
                "start": 2810,
                "end": 2819
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2810,
              "end": 2819
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2824,
                "end": 2835
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2824,
              "end": 2835
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 2840,
                "end": 2851
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2840,
              "end": 2851
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2856,
                "end": 2865
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2856,
              "end": 2865
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2870,
                "end": 2877
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2870,
              "end": 2877
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 2882,
                "end": 2890
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2882,
              "end": 2890
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2895,
                "end": 2907
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2895,
              "end": 2907
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2912,
                "end": 2920
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2912,
              "end": 2920
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 2925,
                "end": 2933
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2925,
              "end": 2933
            }
          }
        ],
        "loc": {
          "start": 2804,
          "end": 2935
        }
      },
      "loc": {
        "start": 2800,
        "end": 2935
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2966,
          "end": 2968
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2966,
        "end": 2968
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2969,
          "end": 2978
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2969,
        "end": 2978
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 3016,
          "end": 3024
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
                "start": 3031,
                "end": 3043
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
                      "start": 3054,
                      "end": 3056
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3054,
                    "end": 3056
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3065,
                      "end": 3073
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3065,
                    "end": 3073
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3082,
                      "end": 3093
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3082,
                    "end": 3093
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3102,
                      "end": 3106
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3102,
                    "end": 3106
                  }
                }
              ],
              "loc": {
                "start": 3044,
                "end": 3112
              }
            },
            "loc": {
              "start": 3031,
              "end": 3112
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3117,
                "end": 3119
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3117,
              "end": 3119
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3124,
                "end": 3134
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3124,
              "end": 3134
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3139,
                "end": 3149
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3139,
              "end": 3149
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 3154,
                "end": 3170
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3154,
              "end": 3170
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 3175,
                "end": 3183
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3175,
              "end": 3183
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3188,
                "end": 3197
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3188,
              "end": 3197
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 3202,
                "end": 3214
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3202,
              "end": 3214
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 3219,
                "end": 3235
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3219,
              "end": 3235
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 3240,
                "end": 3250
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3240,
              "end": 3250
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 3255,
                "end": 3267
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3255,
              "end": 3267
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 3272,
                "end": 3284
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3272,
              "end": 3284
            }
          }
        ],
        "loc": {
          "start": 3025,
          "end": 3286
        }
      },
      "loc": {
        "start": 3016,
        "end": 3286
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3287,
          "end": 3289
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3287,
        "end": 3289
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3290,
          "end": 3300
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3290,
        "end": 3300
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3301,
          "end": 3311
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3301,
        "end": 3311
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3312,
          "end": 3321
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3312,
        "end": 3321
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 3322,
          "end": 3333
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3322,
        "end": 3333
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 3334,
          "end": 3340
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
                "start": 3350,
                "end": 3360
              }
            },
            "directives": [],
            "loc": {
              "start": 3347,
              "end": 3360
            }
          }
        ],
        "loc": {
          "start": 3341,
          "end": 3362
        }
      },
      "loc": {
        "start": 3334,
        "end": 3362
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 3363,
          "end": 3368
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
                  "start": 3382,
                  "end": 3386
                }
              },
              "loc": {
                "start": 3382,
                "end": 3386
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
                      "start": 3400,
                      "end": 3408
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3397,
                    "end": 3408
                  }
                }
              ],
              "loc": {
                "start": 3387,
                "end": 3414
              }
            },
            "loc": {
              "start": 3375,
              "end": 3414
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
                  "start": 3426,
                  "end": 3430
                }
              },
              "loc": {
                "start": 3426,
                "end": 3430
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
                      "start": 3444,
                      "end": 3452
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3441,
                    "end": 3452
                  }
                }
              ],
              "loc": {
                "start": 3431,
                "end": 3458
              }
            },
            "loc": {
              "start": 3419,
              "end": 3458
            }
          }
        ],
        "loc": {
          "start": 3369,
          "end": 3460
        }
      },
      "loc": {
        "start": 3363,
        "end": 3460
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 3461,
          "end": 3472
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3461,
        "end": 3472
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 3473,
          "end": 3487
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3473,
        "end": 3487
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 3488,
          "end": 3493
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3488,
        "end": 3493
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 3494,
          "end": 3503
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3494,
        "end": 3503
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 3504,
          "end": 3508
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
                "start": 3518,
                "end": 3526
              }
            },
            "directives": [],
            "loc": {
              "start": 3515,
              "end": 3526
            }
          }
        ],
        "loc": {
          "start": 3509,
          "end": 3528
        }
      },
      "loc": {
        "start": 3504,
        "end": 3528
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 3529,
          "end": 3543
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3529,
        "end": 3543
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 3544,
          "end": 3549
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3544,
        "end": 3549
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 3550,
          "end": 3553
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
                "start": 3560,
                "end": 3569
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3560,
              "end": 3569
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 3574,
                "end": 3585
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3574,
              "end": 3585
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 3590,
                "end": 3601
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3590,
              "end": 3601
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 3606,
                "end": 3615
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3606,
              "end": 3615
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 3620,
                "end": 3627
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3620,
              "end": 3627
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 3632,
                "end": 3640
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3632,
              "end": 3640
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 3645,
                "end": 3657
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3645,
              "end": 3657
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 3662,
                "end": 3670
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3662,
              "end": 3670
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 3675,
                "end": 3683
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3675,
              "end": 3683
            }
          }
        ],
        "loc": {
          "start": 3554,
          "end": 3685
        }
      },
      "loc": {
        "start": 3550,
        "end": 3685
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3722,
          "end": 3724
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3722,
        "end": 3724
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3725,
          "end": 3734
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3725,
        "end": 3734
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 3774,
          "end": 3786
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
                "start": 3793,
                "end": 3795
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3793,
              "end": 3795
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 3800,
                "end": 3808
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3800,
              "end": 3808
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 3813,
                "end": 3824
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3813,
              "end": 3824
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3829,
                "end": 3833
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3829,
              "end": 3833
            }
          }
        ],
        "loc": {
          "start": 3787,
          "end": 3835
        }
      },
      "loc": {
        "start": 3774,
        "end": 3835
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3836,
          "end": 3838
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3836,
        "end": 3838
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3839,
          "end": 3849
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3839,
        "end": 3849
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3850,
          "end": 3860
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3850,
        "end": 3860
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "createdBy",
        "loc": {
          "start": 3861,
          "end": 3870
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
                "start": 3877,
                "end": 3879
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3877,
              "end": 3879
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3884,
                "end": 3894
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3884,
              "end": 3894
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3899,
                "end": 3909
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3899,
              "end": 3909
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 3914,
                "end": 3925
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3914,
              "end": 3925
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 3930,
                "end": 3936
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3930,
              "end": 3936
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 3941,
                "end": 3946
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3941,
              "end": 3946
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 3951,
                "end": 3971
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3951,
              "end": 3971
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3976,
                "end": 3980
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3976,
              "end": 3980
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 3985,
                "end": 3997
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3985,
              "end": 3997
            }
          }
        ],
        "loc": {
          "start": 3871,
          "end": 3999
        }
      },
      "loc": {
        "start": 3861,
        "end": 3999
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "hasAcceptedAnswer",
        "loc": {
          "start": 4000,
          "end": 4017
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4000,
        "end": 4017
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 4018,
          "end": 4027
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4018,
        "end": 4027
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 4028,
          "end": 4033
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4028,
        "end": 4033
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 4034,
          "end": 4043
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4034,
        "end": 4043
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "answersCount",
        "loc": {
          "start": 4044,
          "end": 4056
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4044,
        "end": 4056
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 4057,
          "end": 4070
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4057,
        "end": 4070
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 4071,
          "end": 4083
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4071,
        "end": 4083
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "forObject",
        "loc": {
          "start": 4084,
          "end": 4093
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
                  "start": 4107,
                  "end": 4110
                }
              },
              "loc": {
                "start": 4107,
                "end": 4110
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
                      "start": 4124,
                      "end": 4131
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4121,
                    "end": 4131
                  }
                }
              ],
              "loc": {
                "start": 4111,
                "end": 4137
              }
            },
            "loc": {
              "start": 4100,
              "end": 4137
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
                  "start": 4149,
                  "end": 4153
                }
              },
              "loc": {
                "start": 4149,
                "end": 4153
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
                      "start": 4167,
                      "end": 4175
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4164,
                    "end": 4175
                  }
                }
              ],
              "loc": {
                "start": 4154,
                "end": 4181
              }
            },
            "loc": {
              "start": 4142,
              "end": 4181
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
                  "start": 4193,
                  "end": 4197
                }
              },
              "loc": {
                "start": 4193,
                "end": 4197
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
                      "start": 4211,
                      "end": 4219
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4208,
                    "end": 4219
                  }
                }
              ],
              "loc": {
                "start": 4198,
                "end": 4225
              }
            },
            "loc": {
              "start": 4186,
              "end": 4225
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
                  "start": 4237,
                  "end": 4244
                }
              },
              "loc": {
                "start": 4237,
                "end": 4244
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
                      "start": 4258,
                      "end": 4269
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4255,
                    "end": 4269
                  }
                }
              ],
              "loc": {
                "start": 4245,
                "end": 4275
              }
            },
            "loc": {
              "start": 4230,
              "end": 4275
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
                  "start": 4287,
                  "end": 4294
                }
              },
              "loc": {
                "start": 4287,
                "end": 4294
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
                      "start": 4308,
                      "end": 4319
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4305,
                    "end": 4319
                  }
                }
              ],
              "loc": {
                "start": 4295,
                "end": 4325
              }
            },
            "loc": {
              "start": 4280,
              "end": 4325
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
                  "start": 4337,
                  "end": 4345
                }
              },
              "loc": {
                "start": 4337,
                "end": 4345
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
                      "start": 4359,
                      "end": 4371
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4356,
                    "end": 4371
                  }
                }
              ],
              "loc": {
                "start": 4346,
                "end": 4377
              }
            },
            "loc": {
              "start": 4330,
              "end": 4377
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
                  "start": 4389,
                  "end": 4393
                }
              },
              "loc": {
                "start": 4389,
                "end": 4393
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
                      "start": 4407,
                      "end": 4415
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4404,
                    "end": 4415
                  }
                }
              ],
              "loc": {
                "start": 4394,
                "end": 4421
              }
            },
            "loc": {
              "start": 4382,
              "end": 4421
            }
          }
        ],
        "loc": {
          "start": 4094,
          "end": 4423
        }
      },
      "loc": {
        "start": 4084,
        "end": 4423
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 4424,
          "end": 4428
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
                "start": 4438,
                "end": 4446
              }
            },
            "directives": [],
            "loc": {
              "start": 4435,
              "end": 4446
            }
          }
        ],
        "loc": {
          "start": 4429,
          "end": 4448
        }
      },
      "loc": {
        "start": 4424,
        "end": 4448
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 4449,
          "end": 4452
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
                "start": 4459,
                "end": 4467
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4459,
              "end": 4467
            }
          }
        ],
        "loc": {
          "start": 4453,
          "end": 4469
        }
      },
      "loc": {
        "start": 4449,
        "end": 4469
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 4507,
          "end": 4515
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
                "start": 4522,
                "end": 4534
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
                      "start": 4545,
                      "end": 4547
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4545,
                    "end": 4547
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 4556,
                      "end": 4564
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4556,
                    "end": 4564
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 4573,
                      "end": 4584
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4573,
                    "end": 4584
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 4593,
                      "end": 4605
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4593,
                    "end": 4605
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4614,
                      "end": 4618
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4614,
                    "end": 4618
                  }
                }
              ],
              "loc": {
                "start": 4535,
                "end": 4624
              }
            },
            "loc": {
              "start": 4522,
              "end": 4624
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 4629,
                "end": 4631
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4629,
              "end": 4631
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4636,
                "end": 4646
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4636,
              "end": 4646
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4651,
                "end": 4661
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4651,
              "end": 4661
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 4666,
                "end": 4677
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4666,
              "end": 4677
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "codeCallData",
              "loc": {
                "start": 4682,
                "end": 4694
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4682,
              "end": 4694
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 4699,
                "end": 4712
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4699,
              "end": 4712
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 4717,
                "end": 4727
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4717,
              "end": 4727
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 4732,
                "end": 4741
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4732,
              "end": 4741
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 4746,
                "end": 4754
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4746,
              "end": 4754
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4759,
                "end": 4768
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4759,
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
              "value": "apiCallData",
              "loc": {
                "start": 4824,
                "end": 4835
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4824,
              "end": 4835
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 4840,
                "end": 4852
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4840,
              "end": 4852
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 4857,
                "end": 4869
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4857,
              "end": 4869
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 4874,
                "end": 4887
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4874,
              "end": 4887
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 4892,
                "end": 4914
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4892,
              "end": 4914
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 4919,
                "end": 4929
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4919,
              "end": 4929
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 4934,
                "end": 4945
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4934,
              "end": 4945
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodesCount",
              "loc": {
                "start": 4950,
                "end": 4960
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4950,
              "end": 4960
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodeLinksCount",
              "loc": {
                "start": 4965,
                "end": 4979
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4965,
              "end": 4979
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
              "loc": {
                "start": 4984,
                "end": 4996
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4984,
              "end": 4996
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 5001,
                "end": 5013
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5001,
              "end": 5013
            }
          }
        ],
        "loc": {
          "start": 4516,
          "end": 5015
        }
      },
      "loc": {
        "start": 4507,
        "end": 5015
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5016,
          "end": 5018
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5016,
        "end": 5018
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 5019,
          "end": 5029
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5019,
        "end": 5029
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 5030,
          "end": 5040
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5030,
        "end": 5040
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 5041,
          "end": 5051
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5041,
        "end": 5051
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5052,
          "end": 5061
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5052,
        "end": 5061
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 5062,
          "end": 5073
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5062,
        "end": 5073
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 5074,
          "end": 5080
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
                "start": 5090,
                "end": 5100
              }
            },
            "directives": [],
            "loc": {
              "start": 5087,
              "end": 5100
            }
          }
        ],
        "loc": {
          "start": 5081,
          "end": 5102
        }
      },
      "loc": {
        "start": 5074,
        "end": 5102
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 5103,
          "end": 5108
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
                  "start": 5122,
                  "end": 5126
                }
              },
              "loc": {
                "start": 5122,
                "end": 5126
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
                      "start": 5140,
                      "end": 5148
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5137,
                    "end": 5148
                  }
                }
              ],
              "loc": {
                "start": 5127,
                "end": 5154
              }
            },
            "loc": {
              "start": 5115,
              "end": 5154
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
                  "start": 5166,
                  "end": 5170
                }
              },
              "loc": {
                "start": 5166,
                "end": 5170
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
                      "start": 5184,
                      "end": 5192
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5181,
                    "end": 5192
                  }
                }
              ],
              "loc": {
                "start": 5171,
                "end": 5198
              }
            },
            "loc": {
              "start": 5159,
              "end": 5198
            }
          }
        ],
        "loc": {
          "start": 5109,
          "end": 5200
        }
      },
      "loc": {
        "start": 5103,
        "end": 5200
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 5201,
          "end": 5212
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5201,
        "end": 5212
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 5213,
          "end": 5227
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5213,
        "end": 5227
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 5228,
          "end": 5233
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5228,
        "end": 5233
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 5234,
          "end": 5243
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5234,
        "end": 5243
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 5244,
          "end": 5248
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
                "start": 5258,
                "end": 5266
              }
            },
            "directives": [],
            "loc": {
              "start": 5255,
              "end": 5266
            }
          }
        ],
        "loc": {
          "start": 5249,
          "end": 5268
        }
      },
      "loc": {
        "start": 5244,
        "end": 5268
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 5269,
          "end": 5283
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5269,
        "end": 5283
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 5284,
          "end": 5289
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5284,
        "end": 5289
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 5290,
          "end": 5293
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
                "start": 5300,
                "end": 5310
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5300,
              "end": 5310
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 5315,
                "end": 5324
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5315,
              "end": 5324
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 5329,
                "end": 5340
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5329,
              "end": 5340
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 5345,
                "end": 5354
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5345,
              "end": 5354
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 5359,
                "end": 5366
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5359,
              "end": 5366
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 5371,
                "end": 5379
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5371,
              "end": 5379
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 5384,
                "end": 5396
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5384,
              "end": 5396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 5401,
                "end": 5409
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5401,
              "end": 5409
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 5414,
                "end": 5422
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5414,
              "end": 5422
            }
          }
        ],
        "loc": {
          "start": 5294,
          "end": 5424
        }
      },
      "loc": {
        "start": 5290,
        "end": 5424
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5461,
          "end": 5463
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5461,
        "end": 5463
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 5464,
          "end": 5474
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5464,
        "end": 5474
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5475,
          "end": 5484
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5475,
        "end": 5484
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 5524,
          "end": 5532
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
                "start": 5539,
                "end": 5551
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
                      "start": 5562,
                      "end": 5564
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5562,
                    "end": 5564
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 5573,
                      "end": 5581
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5573,
                    "end": 5581
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 5590,
                      "end": 5601
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5590,
                    "end": 5601
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 5610,
                      "end": 5622
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5610,
                    "end": 5622
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5631,
                      "end": 5635
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5631,
                    "end": 5635
                  }
                }
              ],
              "loc": {
                "start": 5552,
                "end": 5641
              }
            },
            "loc": {
              "start": 5539,
              "end": 5641
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5646,
                "end": 5648
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5646,
              "end": 5648
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5653,
                "end": 5663
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5653,
              "end": 5663
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5668,
                "end": 5678
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5668,
              "end": 5678
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 5683,
                "end": 5693
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5683,
              "end": 5693
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isFile",
              "loc": {
                "start": 5698,
                "end": 5704
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5698,
              "end": 5704
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 5709,
                "end": 5717
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5709,
              "end": 5717
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5722,
                "end": 5731
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5722,
              "end": 5731
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 5736,
                "end": 5743
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5736,
              "end": 5743
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardType",
              "loc": {
                "start": 5748,
                "end": 5760
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5748,
              "end": 5760
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "props",
              "loc": {
                "start": 5765,
                "end": 5770
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5765,
              "end": 5770
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yup",
              "loc": {
                "start": 5775,
                "end": 5778
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5775,
              "end": 5778
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 5783,
                "end": 5795
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5783,
              "end": 5795
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 5800,
                "end": 5812
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5800,
              "end": 5812
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 5817,
                "end": 5830
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5817,
              "end": 5830
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 5835,
                "end": 5857
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5835,
              "end": 5857
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 5862,
                "end": 5872
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5862,
              "end": 5872
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 5877,
                "end": 5889
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5877,
              "end": 5889
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5894,
                "end": 5897
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
                      "start": 5908,
                      "end": 5918
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5908,
                    "end": 5918
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 5927,
                      "end": 5934
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5927,
                    "end": 5934
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5943,
                      "end": 5952
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5943,
                    "end": 5952
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 5961,
                      "end": 5970
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5961,
                    "end": 5970
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5979,
                      "end": 5988
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5979,
                    "end": 5988
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 5997,
                      "end": 6003
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5997,
                    "end": 6003
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6012,
                      "end": 6019
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6012,
                    "end": 6019
                  }
                }
              ],
              "loc": {
                "start": 5898,
                "end": 6025
              }
            },
            "loc": {
              "start": 5894,
              "end": 6025
            }
          }
        ],
        "loc": {
          "start": 5533,
          "end": 6027
        }
      },
      "loc": {
        "start": 5524,
        "end": 6027
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6028,
          "end": 6030
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6028,
        "end": 6030
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6031,
          "end": 6041
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6031,
        "end": 6041
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 6042,
          "end": 6052
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6042,
        "end": 6052
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6053,
          "end": 6062
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6053,
        "end": 6062
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 6063,
          "end": 6074
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6063,
        "end": 6074
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 6075,
          "end": 6081
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
                "start": 6091,
                "end": 6101
              }
            },
            "directives": [],
            "loc": {
              "start": 6088,
              "end": 6101
            }
          }
        ],
        "loc": {
          "start": 6082,
          "end": 6103
        }
      },
      "loc": {
        "start": 6075,
        "end": 6103
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 6104,
          "end": 6109
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
                  "start": 6123,
                  "end": 6127
                }
              },
              "loc": {
                "start": 6123,
                "end": 6127
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
                      "start": 6141,
                      "end": 6149
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6138,
                    "end": 6149
                  }
                }
              ],
              "loc": {
                "start": 6128,
                "end": 6155
              }
            },
            "loc": {
              "start": 6116,
              "end": 6155
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
                  "start": 6167,
                  "end": 6171
                }
              },
              "loc": {
                "start": 6167,
                "end": 6171
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
                      "start": 6185,
                      "end": 6193
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6182,
                    "end": 6193
                  }
                }
              ],
              "loc": {
                "start": 6172,
                "end": 6199
              }
            },
            "loc": {
              "start": 6160,
              "end": 6199
            }
          }
        ],
        "loc": {
          "start": 6110,
          "end": 6201
        }
      },
      "loc": {
        "start": 6104,
        "end": 6201
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 6202,
          "end": 6213
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6202,
        "end": 6213
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 6214,
          "end": 6228
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6214,
        "end": 6228
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 6229,
          "end": 6234
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6229,
        "end": 6234
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6235,
          "end": 6244
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6235,
        "end": 6244
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 6245,
          "end": 6249
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
                "start": 6259,
                "end": 6267
              }
            },
            "directives": [],
            "loc": {
              "start": 6256,
              "end": 6267
            }
          }
        ],
        "loc": {
          "start": 6250,
          "end": 6269
        }
      },
      "loc": {
        "start": 6245,
        "end": 6269
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 6270,
          "end": 6284
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6270,
        "end": 6284
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 6285,
          "end": 6290
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6285,
        "end": 6290
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6291,
          "end": 6294
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
                "start": 6301,
                "end": 6310
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6301,
              "end": 6310
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
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
              "value": "canTransfer",
              "loc": {
                "start": 6331,
                "end": 6342
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6331,
              "end": 6342
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 6347,
                "end": 6356
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6347,
              "end": 6356
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 6361,
                "end": 6368
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6361,
              "end": 6368
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 6373,
                "end": 6381
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6373,
              "end": 6381
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6386,
                "end": 6398
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6386,
              "end": 6398
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 6403,
                "end": 6411
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6403,
              "end": 6411
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 6416,
                "end": 6424
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6416,
              "end": 6424
            }
          }
        ],
        "loc": {
          "start": 6295,
          "end": 6426
        }
      },
      "loc": {
        "start": 6291,
        "end": 6426
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6465,
          "end": 6467
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6465,
        "end": 6467
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6468,
          "end": 6477
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6468,
        "end": 6477
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6507,
          "end": 6509
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6507,
        "end": 6509
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6510,
          "end": 6520
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6510,
        "end": 6520
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 6521,
          "end": 6524
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6521,
        "end": 6524
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6525,
          "end": 6534
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6525,
        "end": 6534
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 6535,
          "end": 6547
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
                "start": 6554,
                "end": 6556
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6554,
              "end": 6556
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 6561,
                "end": 6569
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6561,
              "end": 6569
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 6574,
                "end": 6585
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6574,
              "end": 6585
            }
          }
        ],
        "loc": {
          "start": 6548,
          "end": 6587
        }
      },
      "loc": {
        "start": 6535,
        "end": 6587
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6588,
          "end": 6591
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
                "start": 6598,
                "end": 6603
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6598,
              "end": 6603
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6608,
                "end": 6620
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6608,
              "end": 6620
            }
          }
        ],
        "loc": {
          "start": 6592,
          "end": 6622
        }
      },
      "loc": {
        "start": 6588,
        "end": 6622
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6654,
          "end": 6656
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6654,
        "end": 6656
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 6657,
          "end": 6668
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6657,
        "end": 6668
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 6669,
          "end": 6675
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6669,
        "end": 6675
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6676,
          "end": 6686
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6676,
        "end": 6686
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 6687,
          "end": 6697
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6687,
        "end": 6697
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isOpenToNewMembers",
        "loc": {
          "start": 6698,
          "end": 6716
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6698,
        "end": 6716
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6717,
          "end": 6726
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6717,
        "end": 6726
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 6727,
          "end": 6740
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6727,
        "end": 6740
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "membersCount",
        "loc": {
          "start": 6741,
          "end": 6753
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6741,
        "end": 6753
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 6754,
          "end": 6766
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6754,
        "end": 6766
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 6767,
          "end": 6779
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6767,
        "end": 6779
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6780,
          "end": 6789
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6780,
        "end": 6789
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 6790,
          "end": 6794
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
                "start": 6804,
                "end": 6812
              }
            },
            "directives": [],
            "loc": {
              "start": 6801,
              "end": 6812
            }
          }
        ],
        "loc": {
          "start": 6795,
          "end": 6814
        }
      },
      "loc": {
        "start": 6790,
        "end": 6814
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 6815,
          "end": 6827
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
                "start": 6834,
                "end": 6836
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6834,
              "end": 6836
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 6841,
                "end": 6849
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6841,
              "end": 6849
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 6854,
                "end": 6857
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6854,
              "end": 6857
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 6862,
                "end": 6866
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6862,
              "end": 6866
            }
          }
        ],
        "loc": {
          "start": 6828,
          "end": 6868
        }
      },
      "loc": {
        "start": 6815,
        "end": 6868
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6869,
          "end": 6872
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
                "start": 6879,
                "end": 6892
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6879,
              "end": 6892
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 6897,
                "end": 6906
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6897,
              "end": 6906
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 6911,
                "end": 6922
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6911,
              "end": 6922
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 6927,
                "end": 6936
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6927,
              "end": 6936
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 6941,
                "end": 6950
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6941,
              "end": 6950
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 6955,
                "end": 6962
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6955,
              "end": 6962
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6967,
                "end": 6979
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6967,
              "end": 6979
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 6984,
                "end": 6992
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6984,
              "end": 6992
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 6997,
                "end": 7011
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
                      "start": 7022,
                      "end": 7024
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7022,
                    "end": 7024
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 7033,
                      "end": 7043
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7033,
                    "end": 7043
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 7052,
                      "end": 7062
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7052,
                    "end": 7062
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 7071,
                      "end": 7078
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7071,
                    "end": 7078
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 7087,
                      "end": 7098
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7087,
                    "end": 7098
                  }
                }
              ],
              "loc": {
                "start": 7012,
                "end": 7104
              }
            },
            "loc": {
              "start": 6997,
              "end": 7104
            }
          }
        ],
        "loc": {
          "start": 6873,
          "end": 7106
        }
      },
      "loc": {
        "start": 6869,
        "end": 7106
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7137,
          "end": 7139
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7137,
        "end": 7139
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7140,
          "end": 7151
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7140,
        "end": 7151
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7152,
          "end": 7158
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7152,
        "end": 7158
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7159,
          "end": 7171
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7159,
        "end": 7171
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7172,
          "end": 7175
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
                "start": 7182,
                "end": 7195
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7182,
              "end": 7195
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 7200,
                "end": 7209
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7200,
              "end": 7209
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 7214,
                "end": 7225
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7214,
              "end": 7225
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 7230,
                "end": 7239
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7230,
              "end": 7239
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7244,
                "end": 7253
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7244,
              "end": 7253
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 7258,
                "end": 7265
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7258,
              "end": 7265
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7270,
                "end": 7282
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7270,
              "end": 7282
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7287,
                "end": 7295
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7287,
              "end": 7295
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 7300,
                "end": 7314
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
                      "start": 7325,
                      "end": 7327
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7325,
                    "end": 7327
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 7336,
                      "end": 7346
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7336,
                    "end": 7346
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 7355,
                      "end": 7365
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7355,
                    "end": 7365
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 7374,
                      "end": 7381
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7374,
                    "end": 7381
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 7390,
                      "end": 7401
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7390,
                    "end": 7401
                  }
                }
              ],
              "loc": {
                "start": 7315,
                "end": 7407
              }
            },
            "loc": {
              "start": 7300,
              "end": 7407
            }
          }
        ],
        "loc": {
          "start": 7176,
          "end": 7409
        }
      },
      "loc": {
        "start": 7172,
        "end": 7409
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 7441,
          "end": 7453
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
                "start": 7460,
                "end": 7462
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7460,
              "end": 7462
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 7467,
                "end": 7475
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7467,
              "end": 7475
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 7480,
                "end": 7483
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7480,
              "end": 7483
            }
          }
        ],
        "loc": {
          "start": 7454,
          "end": 7485
        }
      },
      "loc": {
        "start": 7441,
        "end": 7485
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7486,
          "end": 7488
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7486,
        "end": 7488
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7489,
          "end": 7499
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7489,
        "end": 7499
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7500,
          "end": 7510
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7500,
        "end": 7510
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7511,
          "end": 7522
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7511,
        "end": 7522
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7523,
          "end": 7529
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7523,
        "end": 7529
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7530,
          "end": 7535
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7530,
        "end": 7535
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 7536,
          "end": 7556
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7536,
        "end": 7556
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7557,
          "end": 7561
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7557,
        "end": 7561
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7562,
          "end": 7574
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7562,
        "end": 7574
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7575,
          "end": 7584
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7575,
        "end": 7584
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsReceivedCount",
        "loc": {
          "start": 7585,
          "end": 7605
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7585,
        "end": 7605
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7606,
          "end": 7609
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
                "start": 7616,
                "end": 7625
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7616,
              "end": 7625
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 7630,
                "end": 7639
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7630,
              "end": 7639
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7644,
                "end": 7653
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7644,
              "end": 7653
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7658,
                "end": 7670
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7658,
              "end": 7670
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7675,
                "end": 7683
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7675,
              "end": 7683
            }
          }
        ],
        "loc": {
          "start": 7610,
          "end": 7685
        }
      },
      "loc": {
        "start": 7606,
        "end": 7685
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7716,
          "end": 7718
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7716,
        "end": 7718
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7719,
          "end": 7729
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7719,
        "end": 7729
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7730,
          "end": 7740
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7730,
        "end": 7740
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7741,
          "end": 7752
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7741,
        "end": 7752
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7753,
          "end": 7759
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7753,
        "end": 7759
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7760,
          "end": 7765
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7760,
        "end": 7765
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 7766,
          "end": 7786
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7766,
        "end": 7786
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7787,
          "end": 7791
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7787,
        "end": 7791
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7792,
          "end": 7804
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7792,
        "end": 7804
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
                    "value": "default",
                    "loc": {
                      "start": 1144,
                      "end": 1151
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1144,
                    "end": 1151
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contractType",
                    "loc": {
                      "start": 1156,
                      "end": 1168
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1156,
                    "end": 1168
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "content",
                    "loc": {
                      "start": 1173,
                      "end": 1180
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1173,
                    "end": 1180
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1185,
                      "end": 1197
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1185,
                    "end": 1197
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1202,
                      "end": 1214
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1202,
                    "end": 1214
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 1219,
                      "end": 1232
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1219,
                    "end": 1232
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 1237,
                      "end": 1259
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1237,
                    "end": 1259
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 1264,
                      "end": 1274
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1264,
                    "end": 1274
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1279,
                      "end": 1291
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1279,
                    "end": 1291
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 1296,
                      "end": 1299
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
                            "start": 1310,
                            "end": 1320
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1310,
                          "end": 1320
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 1329,
                            "end": 1336
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1329,
                          "end": 1336
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 1345,
                            "end": 1354
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1345,
                          "end": 1354
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 1363,
                            "end": 1372
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1363,
                          "end": 1372
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 1381,
                            "end": 1390
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1381,
                          "end": 1390
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 1399,
                            "end": 1405
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1399,
                          "end": 1405
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 1414,
                            "end": 1421
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1414,
                          "end": 1421
                        }
                      }
                    ],
                    "loc": {
                      "start": 1300,
                      "end": 1427
                    }
                  },
                  "loc": {
                    "start": 1296,
                    "end": 1427
                  }
                }
              ],
              "loc": {
                "start": 938,
                "end": 1429
              }
            },
            "loc": {
              "start": 929,
              "end": 1429
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1430,
                "end": 1432
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1430,
              "end": 1432
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1433,
                "end": 1443
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1433,
              "end": 1443
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1444,
                "end": 1454
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1444,
              "end": 1454
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1455,
                "end": 1464
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1455,
              "end": 1464
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1465,
                "end": 1476
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1465,
              "end": 1476
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1477,
                "end": 1483
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
                      "start": 1493,
                      "end": 1503
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1490,
                    "end": 1503
                  }
                }
              ],
              "loc": {
                "start": 1484,
                "end": 1505
              }
            },
            "loc": {
              "start": 1477,
              "end": 1505
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1506,
                "end": 1511
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
                        "start": 1525,
                        "end": 1529
                      }
                    },
                    "loc": {
                      "start": 1525,
                      "end": 1529
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
                            "start": 1543,
                            "end": 1551
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1540,
                          "end": 1551
                        }
                      }
                    ],
                    "loc": {
                      "start": 1530,
                      "end": 1557
                    }
                  },
                  "loc": {
                    "start": 1518,
                    "end": 1557
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
                        "start": 1569,
                        "end": 1573
                      }
                    },
                    "loc": {
                      "start": 1569,
                      "end": 1573
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
                            "start": 1587,
                            "end": 1595
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1584,
                          "end": 1595
                        }
                      }
                    ],
                    "loc": {
                      "start": 1574,
                      "end": 1601
                    }
                  },
                  "loc": {
                    "start": 1562,
                    "end": 1601
                  }
                }
              ],
              "loc": {
                "start": 1512,
                "end": 1603
              }
            },
            "loc": {
              "start": 1506,
              "end": 1603
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1604,
                "end": 1615
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1604,
              "end": 1615
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1616,
                "end": 1630
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1616,
              "end": 1630
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1631,
                "end": 1636
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1631,
              "end": 1636
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1637,
                "end": 1646
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1637,
              "end": 1646
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1647,
                "end": 1651
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
                      "start": 1661,
                      "end": 1669
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1658,
                    "end": 1669
                  }
                }
              ],
              "loc": {
                "start": 1652,
                "end": 1671
              }
            },
            "loc": {
              "start": 1647,
              "end": 1671
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1672,
                "end": 1686
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1672,
              "end": 1686
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1687,
                "end": 1692
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1687,
              "end": 1692
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1693,
                "end": 1696
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
                      "start": 1703,
                      "end": 1712
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1703,
                    "end": 1712
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1717,
                      "end": 1728
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1717,
                    "end": 1728
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 1733,
                      "end": 1744
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1733,
                    "end": 1744
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1749,
                      "end": 1758
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1749,
                    "end": 1758
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1763,
                      "end": 1770
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1763,
                    "end": 1770
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1775,
                      "end": 1783
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1775,
                    "end": 1783
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1788,
                      "end": 1800
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1788,
                    "end": 1800
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1805,
                      "end": 1813
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1805,
                    "end": 1813
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1818,
                      "end": 1826
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1818,
                    "end": 1826
                  }
                }
              ],
              "loc": {
                "start": 1697,
                "end": 1828
              }
            },
            "loc": {
              "start": 1693,
              "end": 1828
            }
          }
        ],
        "loc": {
          "start": 927,
          "end": 1830
        }
      },
      "loc": {
        "start": 900,
        "end": 1830
      }
    },
    "Code_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Code_nav",
        "loc": {
          "start": 1840,
          "end": 1848
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Code",
          "loc": {
            "start": 1852,
            "end": 1856
          }
        },
        "loc": {
          "start": 1852,
          "end": 1856
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
                "start": 1859,
                "end": 1861
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1859,
              "end": 1861
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1862,
                "end": 1871
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1862,
              "end": 1871
            }
          }
        ],
        "loc": {
          "start": 1857,
          "end": 1873
        }
      },
      "loc": {
        "start": 1831,
        "end": 1873
      }
    },
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 1883,
          "end": 1893
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 1897,
            "end": 1902
          }
        },
        "loc": {
          "start": 1897,
          "end": 1902
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
                "start": 1905,
                "end": 1907
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1905,
              "end": 1907
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1908,
                "end": 1918
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1908,
              "end": 1918
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1919,
                "end": 1929
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1919,
              "end": 1929
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 1930,
                "end": 1935
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1930,
              "end": 1935
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 1936,
                "end": 1941
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1936,
              "end": 1941
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1942,
                "end": 1947
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
                        "start": 1961,
                        "end": 1965
                      }
                    },
                    "loc": {
                      "start": 1961,
                      "end": 1965
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
                            "start": 1979,
                            "end": 1987
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1976,
                          "end": 1987
                        }
                      }
                    ],
                    "loc": {
                      "start": 1966,
                      "end": 1993
                    }
                  },
                  "loc": {
                    "start": 1954,
                    "end": 1993
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
                        "start": 2005,
                        "end": 2009
                      }
                    },
                    "loc": {
                      "start": 2005,
                      "end": 2009
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
                            "start": 2023,
                            "end": 2031
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2020,
                          "end": 2031
                        }
                      }
                    ],
                    "loc": {
                      "start": 2010,
                      "end": 2037
                    }
                  },
                  "loc": {
                    "start": 1998,
                    "end": 2037
                  }
                }
              ],
              "loc": {
                "start": 1948,
                "end": 2039
              }
            },
            "loc": {
              "start": 1942,
              "end": 2039
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2040,
                "end": 2043
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
                      "start": 2050,
                      "end": 2059
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2050,
                    "end": 2059
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2064,
                      "end": 2073
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2064,
                    "end": 2073
                  }
                }
              ],
              "loc": {
                "start": 2044,
                "end": 2075
              }
            },
            "loc": {
              "start": 2040,
              "end": 2075
            }
          }
        ],
        "loc": {
          "start": 1903,
          "end": 2077
        }
      },
      "loc": {
        "start": 1874,
        "end": 2077
      }
    },
    "Note_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_list",
        "loc": {
          "start": 2087,
          "end": 2096
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 2100,
            "end": 2104
          }
        },
        "loc": {
          "start": 2100,
          "end": 2104
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
                "start": 2107,
                "end": 2115
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
                      "start": 2122,
                      "end": 2134
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
                            "start": 2145,
                            "end": 2147
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2145,
                          "end": 2147
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 2156,
                            "end": 2164
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2156,
                          "end": 2164
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2173,
                            "end": 2184
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2173,
                          "end": 2184
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2193,
                            "end": 2197
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2193,
                          "end": 2197
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pages",
                          "loc": {
                            "start": 2206,
                            "end": 2211
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
                                  "start": 2226,
                                  "end": 2228
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2226,
                                "end": 2228
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "pageIndex",
                                "loc": {
                                  "start": 2241,
                                  "end": 2250
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2241,
                                "end": 2250
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "text",
                                "loc": {
                                  "start": 2263,
                                  "end": 2267
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2263,
                                "end": 2267
                              }
                            }
                          ],
                          "loc": {
                            "start": 2212,
                            "end": 2277
                          }
                        },
                        "loc": {
                          "start": 2206,
                          "end": 2277
                        }
                      }
                    ],
                    "loc": {
                      "start": 2135,
                      "end": 2283
                    }
                  },
                  "loc": {
                    "start": 2122,
                    "end": 2283
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2288,
                      "end": 2290
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2288,
                    "end": 2290
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2295,
                      "end": 2305
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2295,
                    "end": 2305
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2310,
                      "end": 2320
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2310,
                    "end": 2320
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 2325,
                      "end": 2333
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2325,
                    "end": 2333
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 2338,
                      "end": 2347
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2338,
                    "end": 2347
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 2352,
                      "end": 2364
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2352,
                    "end": 2364
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 2369,
                      "end": 2381
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2369,
                    "end": 2381
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 2386,
                      "end": 2398
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2386,
                    "end": 2398
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 2403,
                      "end": 2406
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
                            "start": 2417,
                            "end": 2427
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2417,
                          "end": 2427
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 2436,
                            "end": 2443
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2436,
                          "end": 2443
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 2452,
                            "end": 2461
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2452,
                          "end": 2461
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 2470,
                            "end": 2479
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2470,
                          "end": 2479
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 2488,
                            "end": 2497
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2488,
                          "end": 2497
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 2506,
                            "end": 2512
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2506,
                          "end": 2512
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 2521,
                            "end": 2528
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2521,
                          "end": 2528
                        }
                      }
                    ],
                    "loc": {
                      "start": 2407,
                      "end": 2534
                    }
                  },
                  "loc": {
                    "start": 2403,
                    "end": 2534
                  }
                }
              ],
              "loc": {
                "start": 2116,
                "end": 2536
              }
            },
            "loc": {
              "start": 2107,
              "end": 2536
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2537,
                "end": 2539
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2537,
              "end": 2539
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2540,
                "end": 2550
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2540,
              "end": 2550
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2551,
                "end": 2561
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2551,
              "end": 2561
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2562,
                "end": 2571
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2562,
              "end": 2571
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 2572,
                "end": 2583
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2572,
              "end": 2583
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 2584,
                "end": 2590
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
                      "start": 2600,
                      "end": 2610
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2597,
                    "end": 2610
                  }
                }
              ],
              "loc": {
                "start": 2591,
                "end": 2612
              }
            },
            "loc": {
              "start": 2584,
              "end": 2612
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 2613,
                "end": 2618
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
                        "start": 2632,
                        "end": 2636
                      }
                    },
                    "loc": {
                      "start": 2632,
                      "end": 2636
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
                            "start": 2650,
                            "end": 2658
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2647,
                          "end": 2658
                        }
                      }
                    ],
                    "loc": {
                      "start": 2637,
                      "end": 2664
                    }
                  },
                  "loc": {
                    "start": 2625,
                    "end": 2664
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
                        "start": 2676,
                        "end": 2680
                      }
                    },
                    "loc": {
                      "start": 2676,
                      "end": 2680
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
                            "start": 2694,
                            "end": 2702
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2691,
                          "end": 2702
                        }
                      }
                    ],
                    "loc": {
                      "start": 2681,
                      "end": 2708
                    }
                  },
                  "loc": {
                    "start": 2669,
                    "end": 2708
                  }
                }
              ],
              "loc": {
                "start": 2619,
                "end": 2710
              }
            },
            "loc": {
              "start": 2613,
              "end": 2710
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 2711,
                "end": 2722
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2711,
              "end": 2722
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 2723,
                "end": 2737
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2723,
              "end": 2737
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 2738,
                "end": 2743
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2738,
              "end": 2743
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2744,
                "end": 2753
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2744,
              "end": 2753
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2754,
                "end": 2758
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
                      "start": 2768,
                      "end": 2776
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2765,
                    "end": 2776
                  }
                }
              ],
              "loc": {
                "start": 2759,
                "end": 2778
              }
            },
            "loc": {
              "start": 2754,
              "end": 2778
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 2779,
                "end": 2793
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2779,
              "end": 2793
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 2794,
                "end": 2799
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2794,
              "end": 2799
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2800,
                "end": 2803
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
                      "start": 2810,
                      "end": 2819
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2810,
                    "end": 2819
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2824,
                      "end": 2835
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2824,
                    "end": 2835
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 2840,
                      "end": 2851
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2840,
                    "end": 2851
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2856,
                      "end": 2865
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2856,
                    "end": 2865
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2870,
                      "end": 2877
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2870,
                    "end": 2877
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 2882,
                      "end": 2890
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2882,
                    "end": 2890
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2895,
                      "end": 2907
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2895,
                    "end": 2907
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2912,
                      "end": 2920
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2912,
                    "end": 2920
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 2925,
                      "end": 2933
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2925,
                    "end": 2933
                  }
                }
              ],
              "loc": {
                "start": 2804,
                "end": 2935
              }
            },
            "loc": {
              "start": 2800,
              "end": 2935
            }
          }
        ],
        "loc": {
          "start": 2105,
          "end": 2937
        }
      },
      "loc": {
        "start": 2078,
        "end": 2937
      }
    },
    "Note_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_nav",
        "loc": {
          "start": 2947,
          "end": 2955
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 2959,
            "end": 2963
          }
        },
        "loc": {
          "start": 2959,
          "end": 2963
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
                "start": 2966,
                "end": 2968
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2966,
              "end": 2968
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2969,
                "end": 2978
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2969,
              "end": 2978
            }
          }
        ],
        "loc": {
          "start": 2964,
          "end": 2980
        }
      },
      "loc": {
        "start": 2938,
        "end": 2980
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 2990,
          "end": 3002
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3006,
            "end": 3013
          }
        },
        "loc": {
          "start": 3006,
          "end": 3013
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
                "start": 3016,
                "end": 3024
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
                      "start": 3031,
                      "end": 3043
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
                            "start": 3054,
                            "end": 3056
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3054,
                          "end": 3056
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 3065,
                            "end": 3073
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3065,
                          "end": 3073
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 3082,
                            "end": 3093
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3082,
                          "end": 3093
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 3102,
                            "end": 3106
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3102,
                          "end": 3106
                        }
                      }
                    ],
                    "loc": {
                      "start": 3044,
                      "end": 3112
                    }
                  },
                  "loc": {
                    "start": 3031,
                    "end": 3112
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3117,
                      "end": 3119
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3117,
                    "end": 3119
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3124,
                      "end": 3134
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3124,
                    "end": 3134
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3139,
                      "end": 3149
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3139,
                    "end": 3149
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 3154,
                      "end": 3170
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3154,
                    "end": 3170
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 3175,
                      "end": 3183
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3175,
                    "end": 3183
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 3188,
                      "end": 3197
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3188,
                    "end": 3197
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 3202,
                      "end": 3214
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3202,
                    "end": 3214
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 3219,
                      "end": 3235
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3219,
                    "end": 3235
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 3240,
                      "end": 3250
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3240,
                    "end": 3250
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 3255,
                      "end": 3267
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3255,
                    "end": 3267
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 3272,
                      "end": 3284
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3272,
                    "end": 3284
                  }
                }
              ],
              "loc": {
                "start": 3025,
                "end": 3286
              }
            },
            "loc": {
              "start": 3016,
              "end": 3286
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3287,
                "end": 3289
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3287,
              "end": 3289
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3290,
                "end": 3300
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3290,
              "end": 3300
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3301,
                "end": 3311
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3301,
              "end": 3311
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3312,
                "end": 3321
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3312,
              "end": 3321
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 3322,
                "end": 3333
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3322,
              "end": 3333
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 3334,
                "end": 3340
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
                      "start": 3350,
                      "end": 3360
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3347,
                    "end": 3360
                  }
                }
              ],
              "loc": {
                "start": 3341,
                "end": 3362
              }
            },
            "loc": {
              "start": 3334,
              "end": 3362
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 3363,
                "end": 3368
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
                        "start": 3382,
                        "end": 3386
                      }
                    },
                    "loc": {
                      "start": 3382,
                      "end": 3386
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
                            "start": 3400,
                            "end": 3408
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3397,
                          "end": 3408
                        }
                      }
                    ],
                    "loc": {
                      "start": 3387,
                      "end": 3414
                    }
                  },
                  "loc": {
                    "start": 3375,
                    "end": 3414
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
                        "start": 3426,
                        "end": 3430
                      }
                    },
                    "loc": {
                      "start": 3426,
                      "end": 3430
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
                            "start": 3444,
                            "end": 3452
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3441,
                          "end": 3452
                        }
                      }
                    ],
                    "loc": {
                      "start": 3431,
                      "end": 3458
                    }
                  },
                  "loc": {
                    "start": 3419,
                    "end": 3458
                  }
                }
              ],
              "loc": {
                "start": 3369,
                "end": 3460
              }
            },
            "loc": {
              "start": 3363,
              "end": 3460
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 3461,
                "end": 3472
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3461,
              "end": 3472
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 3473,
                "end": 3487
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3473,
              "end": 3487
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 3488,
                "end": 3493
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3488,
              "end": 3493
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 3494,
                "end": 3503
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3494,
              "end": 3503
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 3504,
                "end": 3508
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
                      "start": 3518,
                      "end": 3526
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3515,
                    "end": 3526
                  }
                }
              ],
              "loc": {
                "start": 3509,
                "end": 3528
              }
            },
            "loc": {
              "start": 3504,
              "end": 3528
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 3529,
                "end": 3543
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3529,
              "end": 3543
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 3544,
                "end": 3549
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3544,
              "end": 3549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 3550,
                "end": 3553
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
                      "start": 3560,
                      "end": 3569
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3560,
                    "end": 3569
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 3574,
                      "end": 3585
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3574,
                    "end": 3585
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 3590,
                      "end": 3601
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3590,
                    "end": 3601
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 3606,
                      "end": 3615
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3606,
                    "end": 3615
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 3620,
                      "end": 3627
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3620,
                    "end": 3627
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 3632,
                      "end": 3640
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3632,
                    "end": 3640
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 3645,
                      "end": 3657
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3645,
                    "end": 3657
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 3662,
                      "end": 3670
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3662,
                    "end": 3670
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 3675,
                      "end": 3683
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3675,
                    "end": 3683
                  }
                }
              ],
              "loc": {
                "start": 3554,
                "end": 3685
              }
            },
            "loc": {
              "start": 3550,
              "end": 3685
            }
          }
        ],
        "loc": {
          "start": 3014,
          "end": 3687
        }
      },
      "loc": {
        "start": 2981,
        "end": 3687
      }
    },
    "Project_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_nav",
        "loc": {
          "start": 3697,
          "end": 3708
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3712,
            "end": 3719
          }
        },
        "loc": {
          "start": 3712,
          "end": 3719
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
                "start": 3722,
                "end": 3724
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3722,
              "end": 3724
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3725,
                "end": 3734
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3725,
              "end": 3734
            }
          }
        ],
        "loc": {
          "start": 3720,
          "end": 3736
        }
      },
      "loc": {
        "start": 3688,
        "end": 3736
      }
    },
    "Question_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Question_list",
        "loc": {
          "start": 3746,
          "end": 3759
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Question",
          "loc": {
            "start": 3763,
            "end": 3771
          }
        },
        "loc": {
          "start": 3763,
          "end": 3771
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
                "start": 3774,
                "end": 3786
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
                      "start": 3793,
                      "end": 3795
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3793,
                    "end": 3795
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3800,
                      "end": 3808
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3800,
                    "end": 3808
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3813,
                      "end": 3824
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3813,
                    "end": 3824
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3829,
                      "end": 3833
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3829,
                    "end": 3833
                  }
                }
              ],
              "loc": {
                "start": 3787,
                "end": 3835
              }
            },
            "loc": {
              "start": 3774,
              "end": 3835
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3836,
                "end": 3838
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3836,
              "end": 3838
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3839,
                "end": 3849
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3839,
              "end": 3849
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3850,
                "end": 3860
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3850,
              "end": 3860
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "createdBy",
              "loc": {
                "start": 3861,
                "end": 3870
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
                      "start": 3877,
                      "end": 3879
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3877,
                    "end": 3879
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3884,
                      "end": 3894
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3884,
                    "end": 3894
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3899,
                      "end": 3909
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3899,
                    "end": 3909
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
                    "loc": {
                      "start": 3914,
                      "end": 3925
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3914,
                    "end": 3925
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 3930,
                      "end": 3936
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3930,
                    "end": 3936
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBot",
                    "loc": {
                      "start": 3941,
                      "end": 3946
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3941,
                    "end": 3946
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBotDepictingPerson",
                    "loc": {
                      "start": 3951,
                      "end": 3971
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3951,
                    "end": 3971
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3976,
                      "end": 3980
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3976,
                    "end": 3980
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 3985,
                      "end": 3997
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3985,
                    "end": 3997
                  }
                }
              ],
              "loc": {
                "start": 3871,
                "end": 3999
              }
            },
            "loc": {
              "start": 3861,
              "end": 3999
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "hasAcceptedAnswer",
              "loc": {
                "start": 4000,
                "end": 4017
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4000,
              "end": 4017
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4018,
                "end": 4027
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4018,
              "end": 4027
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 4028,
                "end": 4033
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4028,
              "end": 4033
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 4034,
                "end": 4043
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4034,
              "end": 4043
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "answersCount",
              "loc": {
                "start": 4044,
                "end": 4056
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4044,
              "end": 4056
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 4057,
                "end": 4070
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4057,
              "end": 4070
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 4071,
                "end": 4083
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4071,
              "end": 4083
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forObject",
              "loc": {
                "start": 4084,
                "end": 4093
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
                        "start": 4107,
                        "end": 4110
                      }
                    },
                    "loc": {
                      "start": 4107,
                      "end": 4110
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
                            "start": 4124,
                            "end": 4131
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4121,
                          "end": 4131
                        }
                      }
                    ],
                    "loc": {
                      "start": 4111,
                      "end": 4137
                    }
                  },
                  "loc": {
                    "start": 4100,
                    "end": 4137
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
                        "start": 4149,
                        "end": 4153
                      }
                    },
                    "loc": {
                      "start": 4149,
                      "end": 4153
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
                            "start": 4167,
                            "end": 4175
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4164,
                          "end": 4175
                        }
                      }
                    ],
                    "loc": {
                      "start": 4154,
                      "end": 4181
                    }
                  },
                  "loc": {
                    "start": 4142,
                    "end": 4181
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
                        "start": 4193,
                        "end": 4197
                      }
                    },
                    "loc": {
                      "start": 4193,
                      "end": 4197
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
                            "start": 4211,
                            "end": 4219
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4208,
                          "end": 4219
                        }
                      }
                    ],
                    "loc": {
                      "start": 4198,
                      "end": 4225
                    }
                  },
                  "loc": {
                    "start": 4186,
                    "end": 4225
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
                        "start": 4237,
                        "end": 4244
                      }
                    },
                    "loc": {
                      "start": 4237,
                      "end": 4244
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
                            "start": 4258,
                            "end": 4269
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4255,
                          "end": 4269
                        }
                      }
                    ],
                    "loc": {
                      "start": 4245,
                      "end": 4275
                    }
                  },
                  "loc": {
                    "start": 4230,
                    "end": 4275
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
                        "start": 4287,
                        "end": 4294
                      }
                    },
                    "loc": {
                      "start": 4287,
                      "end": 4294
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
                            "start": 4308,
                            "end": 4319
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4305,
                          "end": 4319
                        }
                      }
                    ],
                    "loc": {
                      "start": 4295,
                      "end": 4325
                    }
                  },
                  "loc": {
                    "start": 4280,
                    "end": 4325
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
                        "start": 4337,
                        "end": 4345
                      }
                    },
                    "loc": {
                      "start": 4337,
                      "end": 4345
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
                            "start": 4359,
                            "end": 4371
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4356,
                          "end": 4371
                        }
                      }
                    ],
                    "loc": {
                      "start": 4346,
                      "end": 4377
                    }
                  },
                  "loc": {
                    "start": 4330,
                    "end": 4377
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
                        "start": 4389,
                        "end": 4393
                      }
                    },
                    "loc": {
                      "start": 4389,
                      "end": 4393
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
                            "start": 4407,
                            "end": 4415
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4404,
                          "end": 4415
                        }
                      }
                    ],
                    "loc": {
                      "start": 4394,
                      "end": 4421
                    }
                  },
                  "loc": {
                    "start": 4382,
                    "end": 4421
                  }
                }
              ],
              "loc": {
                "start": 4094,
                "end": 4423
              }
            },
            "loc": {
              "start": 4084,
              "end": 4423
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 4424,
                "end": 4428
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
                      "start": 4438,
                      "end": 4446
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4435,
                    "end": 4446
                  }
                }
              ],
              "loc": {
                "start": 4429,
                "end": 4448
              }
            },
            "loc": {
              "start": 4424,
              "end": 4448
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 4449,
                "end": 4452
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
                      "start": 4459,
                      "end": 4467
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4459,
                    "end": 4467
                  }
                }
              ],
              "loc": {
                "start": 4453,
                "end": 4469
              }
            },
            "loc": {
              "start": 4449,
              "end": 4469
            }
          }
        ],
        "loc": {
          "start": 3772,
          "end": 4471
        }
      },
      "loc": {
        "start": 3737,
        "end": 4471
      }
    },
    "Routine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_list",
        "loc": {
          "start": 4481,
          "end": 4493
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 4497,
            "end": 4504
          }
        },
        "loc": {
          "start": 4497,
          "end": 4504
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
                "start": 4507,
                "end": 4515
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
                      "start": 4522,
                      "end": 4534
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
                            "start": 4545,
                            "end": 4547
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4545,
                          "end": 4547
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 4556,
                            "end": 4564
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4556,
                          "end": 4564
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 4573,
                            "end": 4584
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4573,
                          "end": 4584
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 4593,
                            "end": 4605
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4593,
                          "end": 4605
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4614,
                            "end": 4618
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4614,
                          "end": 4618
                        }
                      }
                    ],
                    "loc": {
                      "start": 4535,
                      "end": 4624
                    }
                  },
                  "loc": {
                    "start": 4522,
                    "end": 4624
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4629,
                      "end": 4631
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4629,
                    "end": 4631
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4636,
                      "end": 4646
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4636,
                    "end": 4646
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4651,
                      "end": 4661
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4651,
                    "end": 4661
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 4666,
                      "end": 4677
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4666,
                    "end": 4677
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "codeCallData",
                    "loc": {
                      "start": 4682,
                      "end": 4694
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4682,
                    "end": 4694
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 4699,
                      "end": 4712
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4699,
                    "end": 4712
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 4717,
                      "end": 4727
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4717,
                    "end": 4727
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 4732,
                      "end": 4741
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4732,
                    "end": 4741
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 4746,
                      "end": 4754
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4746,
                    "end": 4754
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 4759,
                      "end": 4768
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4759,
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
                    "value": "apiCallData",
                    "loc": {
                      "start": 4824,
                      "end": 4835
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4824,
                    "end": 4835
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 4840,
                      "end": 4852
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4840,
                    "end": 4852
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 4857,
                      "end": 4869
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4857,
                    "end": 4869
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 4874,
                      "end": 4887
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4874,
                    "end": 4887
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 4892,
                      "end": 4914
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4892,
                    "end": 4914
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 4919,
                      "end": 4929
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4919,
                    "end": 4929
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 4934,
                      "end": 4945
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4934,
                    "end": 4945
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 4950,
                      "end": 4960
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4950,
                    "end": 4960
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 4965,
                      "end": 4979
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4965,
                    "end": 4979
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 4984,
                      "end": 4996
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4984,
                    "end": 4996
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 5001,
                      "end": 5013
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5001,
                    "end": 5013
                  }
                }
              ],
              "loc": {
                "start": 4516,
                "end": 5015
              }
            },
            "loc": {
              "start": 4507,
              "end": 5015
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5016,
                "end": 5018
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5016,
              "end": 5018
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5019,
                "end": 5029
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5019,
              "end": 5029
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5030,
                "end": 5040
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5030,
              "end": 5040
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 5041,
                "end": 5051
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5041,
              "end": 5051
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5052,
                "end": 5061
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5052,
              "end": 5061
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 5062,
                "end": 5073
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5062,
              "end": 5073
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 5074,
                "end": 5080
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
                      "start": 5090,
                      "end": 5100
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5087,
                    "end": 5100
                  }
                }
              ],
              "loc": {
                "start": 5081,
                "end": 5102
              }
            },
            "loc": {
              "start": 5074,
              "end": 5102
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 5103,
                "end": 5108
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
                        "start": 5122,
                        "end": 5126
                      }
                    },
                    "loc": {
                      "start": 5122,
                      "end": 5126
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
                            "start": 5140,
                            "end": 5148
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5137,
                          "end": 5148
                        }
                      }
                    ],
                    "loc": {
                      "start": 5127,
                      "end": 5154
                    }
                  },
                  "loc": {
                    "start": 5115,
                    "end": 5154
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
                        "start": 5166,
                        "end": 5170
                      }
                    },
                    "loc": {
                      "start": 5166,
                      "end": 5170
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
                            "start": 5184,
                            "end": 5192
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5181,
                          "end": 5192
                        }
                      }
                    ],
                    "loc": {
                      "start": 5171,
                      "end": 5198
                    }
                  },
                  "loc": {
                    "start": 5159,
                    "end": 5198
                  }
                }
              ],
              "loc": {
                "start": 5109,
                "end": 5200
              }
            },
            "loc": {
              "start": 5103,
              "end": 5200
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 5201,
                "end": 5212
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5201,
              "end": 5212
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 5213,
                "end": 5227
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5213,
              "end": 5227
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 5228,
                "end": 5233
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5228,
              "end": 5233
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 5234,
                "end": 5243
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5234,
              "end": 5243
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 5244,
                "end": 5248
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
                      "start": 5258,
                      "end": 5266
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5255,
                    "end": 5266
                  }
                }
              ],
              "loc": {
                "start": 5249,
                "end": 5268
              }
            },
            "loc": {
              "start": 5244,
              "end": 5268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 5269,
                "end": 5283
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5269,
              "end": 5283
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 5284,
                "end": 5289
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5284,
              "end": 5289
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5290,
                "end": 5293
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
                      "start": 5300,
                      "end": 5310
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5300,
                    "end": 5310
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5315,
                      "end": 5324
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5315,
                    "end": 5324
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 5329,
                      "end": 5340
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5329,
                    "end": 5340
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5345,
                      "end": 5354
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5345,
                    "end": 5354
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5359,
                      "end": 5366
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5359,
                    "end": 5366
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 5371,
                      "end": 5379
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5371,
                    "end": 5379
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 5384,
                      "end": 5396
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5384,
                    "end": 5396
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 5401,
                      "end": 5409
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5401,
                    "end": 5409
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 5414,
                      "end": 5422
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5414,
                    "end": 5422
                  }
                }
              ],
              "loc": {
                "start": 5294,
                "end": 5424
              }
            },
            "loc": {
              "start": 5290,
              "end": 5424
            }
          }
        ],
        "loc": {
          "start": 4505,
          "end": 5426
        }
      },
      "loc": {
        "start": 4472,
        "end": 5426
      }
    },
    "Routine_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_nav",
        "loc": {
          "start": 5436,
          "end": 5447
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 5451,
            "end": 5458
          }
        },
        "loc": {
          "start": 5451,
          "end": 5458
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
                "start": 5461,
                "end": 5463
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5461,
              "end": 5463
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 5464,
                "end": 5474
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5464,
              "end": 5474
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5475,
                "end": 5484
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5475,
              "end": 5484
            }
          }
        ],
        "loc": {
          "start": 5459,
          "end": 5486
        }
      },
      "loc": {
        "start": 5427,
        "end": 5486
      }
    },
    "Standard_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_list",
        "loc": {
          "start": 5496,
          "end": 5509
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 5513,
            "end": 5521
          }
        },
        "loc": {
          "start": 5513,
          "end": 5521
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
                "start": 5524,
                "end": 5532
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
                      "start": 5539,
                      "end": 5551
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
                            "start": 5562,
                            "end": 5564
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5562,
                          "end": 5564
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5573,
                            "end": 5581
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5573,
                          "end": 5581
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5590,
                            "end": 5601
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5590,
                          "end": 5601
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 5610,
                            "end": 5622
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5610,
                          "end": 5622
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5631,
                            "end": 5635
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5631,
                          "end": 5635
                        }
                      }
                    ],
                    "loc": {
                      "start": 5552,
                      "end": 5641
                    }
                  },
                  "loc": {
                    "start": 5539,
                    "end": 5641
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5646,
                      "end": 5648
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5646,
                    "end": 5648
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 5653,
                      "end": 5663
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5653,
                    "end": 5663
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 5668,
                      "end": 5678
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5668,
                    "end": 5678
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 5683,
                      "end": 5693
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5683,
                    "end": 5693
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isFile",
                    "loc": {
                      "start": 5698,
                      "end": 5704
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5698,
                    "end": 5704
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5709,
                      "end": 5717
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5709,
                    "end": 5717
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5722,
                      "end": 5731
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5722,
                    "end": 5731
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 5736,
                      "end": 5743
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5736,
                    "end": 5743
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardType",
                    "loc": {
                      "start": 5748,
                      "end": 5760
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5748,
                    "end": 5760
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "props",
                    "loc": {
                      "start": 5765,
                      "end": 5770
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5765,
                    "end": 5770
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yup",
                    "loc": {
                      "start": 5775,
                      "end": 5778
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5775,
                    "end": 5778
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 5783,
                      "end": 5795
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5783,
                    "end": 5795
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 5800,
                      "end": 5812
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5800,
                    "end": 5812
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 5817,
                      "end": 5830
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5817,
                    "end": 5830
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 5835,
                      "end": 5857
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5835,
                    "end": 5857
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 5862,
                      "end": 5872
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5862,
                    "end": 5872
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 5877,
                      "end": 5889
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5877,
                    "end": 5889
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 5894,
                      "end": 5897
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
                            "start": 5908,
                            "end": 5918
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5908,
                          "end": 5918
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 5927,
                            "end": 5934
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5927,
                          "end": 5934
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 5943,
                            "end": 5952
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5943,
                          "end": 5952
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 5961,
                            "end": 5970
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5961,
                          "end": 5970
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 5979,
                            "end": 5988
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5979,
                          "end": 5988
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 5997,
                            "end": 6003
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5997,
                          "end": 6003
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 6012,
                            "end": 6019
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6012,
                          "end": 6019
                        }
                      }
                    ],
                    "loc": {
                      "start": 5898,
                      "end": 6025
                    }
                  },
                  "loc": {
                    "start": 5894,
                    "end": 6025
                  }
                }
              ],
              "loc": {
                "start": 5533,
                "end": 6027
              }
            },
            "loc": {
              "start": 5524,
              "end": 6027
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6028,
                "end": 6030
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6028,
              "end": 6030
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6031,
                "end": 6041
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6031,
              "end": 6041
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6042,
                "end": 6052
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6042,
              "end": 6052
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6053,
                "end": 6062
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6053,
              "end": 6062
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 6063,
                "end": 6074
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6063,
              "end": 6074
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 6075,
                "end": 6081
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
                      "start": 6091,
                      "end": 6101
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6088,
                    "end": 6101
                  }
                }
              ],
              "loc": {
                "start": 6082,
                "end": 6103
              }
            },
            "loc": {
              "start": 6075,
              "end": 6103
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 6104,
                "end": 6109
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
                        "start": 6123,
                        "end": 6127
                      }
                    },
                    "loc": {
                      "start": 6123,
                      "end": 6127
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
                            "start": 6141,
                            "end": 6149
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6138,
                          "end": 6149
                        }
                      }
                    ],
                    "loc": {
                      "start": 6128,
                      "end": 6155
                    }
                  },
                  "loc": {
                    "start": 6116,
                    "end": 6155
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
                        "start": 6167,
                        "end": 6171
                      }
                    },
                    "loc": {
                      "start": 6167,
                      "end": 6171
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
                            "start": 6185,
                            "end": 6193
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6182,
                          "end": 6193
                        }
                      }
                    ],
                    "loc": {
                      "start": 6172,
                      "end": 6199
                    }
                  },
                  "loc": {
                    "start": 6160,
                    "end": 6199
                  }
                }
              ],
              "loc": {
                "start": 6110,
                "end": 6201
              }
            },
            "loc": {
              "start": 6104,
              "end": 6201
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 6202,
                "end": 6213
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6202,
              "end": 6213
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 6214,
                "end": 6228
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6214,
              "end": 6228
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 6229,
                "end": 6234
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6229,
              "end": 6234
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6235,
                "end": 6244
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6235,
              "end": 6244
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 6245,
                "end": 6249
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
                      "start": 6259,
                      "end": 6267
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6256,
                    "end": 6267
                  }
                }
              ],
              "loc": {
                "start": 6250,
                "end": 6269
              }
            },
            "loc": {
              "start": 6245,
              "end": 6269
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 6270,
                "end": 6284
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6270,
              "end": 6284
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 6285,
                "end": 6290
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6285,
              "end": 6290
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6291,
                "end": 6294
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
                      "start": 6301,
                      "end": 6310
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6301,
                    "end": 6310
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
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
                    "value": "canTransfer",
                    "loc": {
                      "start": 6331,
                      "end": 6342
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6331,
                    "end": 6342
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6347,
                      "end": 6356
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6347,
                    "end": 6356
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6361,
                      "end": 6368
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6361,
                    "end": 6368
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 6373,
                      "end": 6381
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6373,
                    "end": 6381
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6386,
                      "end": 6398
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6386,
                    "end": 6398
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 6403,
                      "end": 6411
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6403,
                    "end": 6411
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 6416,
                      "end": 6424
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6416,
                    "end": 6424
                  }
                }
              ],
              "loc": {
                "start": 6295,
                "end": 6426
              }
            },
            "loc": {
              "start": 6291,
              "end": 6426
            }
          }
        ],
        "loc": {
          "start": 5522,
          "end": 6428
        }
      },
      "loc": {
        "start": 5487,
        "end": 6428
      }
    },
    "Standard_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_nav",
        "loc": {
          "start": 6438,
          "end": 6450
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 6454,
            "end": 6462
          }
        },
        "loc": {
          "start": 6454,
          "end": 6462
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
                "start": 6465,
                "end": 6467
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6465,
              "end": 6467
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6468,
                "end": 6477
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6468,
              "end": 6477
            }
          }
        ],
        "loc": {
          "start": 6463,
          "end": 6479
        }
      },
      "loc": {
        "start": 6429,
        "end": 6479
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 6489,
          "end": 6497
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 6501,
            "end": 6504
          }
        },
        "loc": {
          "start": 6501,
          "end": 6504
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
                "start": 6507,
                "end": 6509
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6507,
              "end": 6509
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6510,
                "end": 6520
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6510,
              "end": 6520
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 6521,
                "end": 6524
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6521,
              "end": 6524
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6525,
                "end": 6534
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6525,
              "end": 6534
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 6535,
                "end": 6547
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
                      "start": 6554,
                      "end": 6556
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6554,
                    "end": 6556
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6561,
                      "end": 6569
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6561,
                    "end": 6569
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 6574,
                      "end": 6585
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6574,
                    "end": 6585
                  }
                }
              ],
              "loc": {
                "start": 6548,
                "end": 6587
              }
            },
            "loc": {
              "start": 6535,
              "end": 6587
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6588,
                "end": 6591
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
                      "start": 6598,
                      "end": 6603
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6598,
                    "end": 6603
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6608,
                      "end": 6620
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6608,
                    "end": 6620
                  }
                }
              ],
              "loc": {
                "start": 6592,
                "end": 6622
              }
            },
            "loc": {
              "start": 6588,
              "end": 6622
            }
          }
        ],
        "loc": {
          "start": 6505,
          "end": 6624
        }
      },
      "loc": {
        "start": 6480,
        "end": 6624
      }
    },
    "Team_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_list",
        "loc": {
          "start": 6634,
          "end": 6643
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 6647,
            "end": 6651
          }
        },
        "loc": {
          "start": 6647,
          "end": 6651
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
                "start": 6654,
                "end": 6656
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6654,
              "end": 6656
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 6657,
                "end": 6668
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6657,
              "end": 6668
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 6669,
                "end": 6675
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6669,
              "end": 6675
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6676,
                "end": 6686
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6676,
              "end": 6686
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6687,
                "end": 6697
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6687,
              "end": 6697
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOpenToNewMembers",
              "loc": {
                "start": 6698,
                "end": 6716
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6698,
              "end": 6716
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6717,
                "end": 6726
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6717,
              "end": 6726
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 6727,
                "end": 6740
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6727,
              "end": 6740
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membersCount",
              "loc": {
                "start": 6741,
                "end": 6753
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6741,
              "end": 6753
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 6754,
                "end": 6766
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6754,
              "end": 6766
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 6767,
                "end": 6779
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6767,
              "end": 6779
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6780,
                "end": 6789
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6780,
              "end": 6789
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 6790,
                "end": 6794
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
                      "start": 6804,
                      "end": 6812
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6801,
                    "end": 6812
                  }
                }
              ],
              "loc": {
                "start": 6795,
                "end": 6814
              }
            },
            "loc": {
              "start": 6790,
              "end": 6814
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 6815,
                "end": 6827
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
                      "start": 6834,
                      "end": 6836
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6834,
                    "end": 6836
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6841,
                      "end": 6849
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6841,
                    "end": 6849
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 6854,
                      "end": 6857
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6854,
                    "end": 6857
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6862,
                      "end": 6866
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6862,
                    "end": 6866
                  }
                }
              ],
              "loc": {
                "start": 6828,
                "end": 6868
              }
            },
            "loc": {
              "start": 6815,
              "end": 6868
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6869,
                "end": 6872
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
                      "start": 6879,
                      "end": 6892
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6879,
                    "end": 6892
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6897,
                      "end": 6906
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6897,
                    "end": 6906
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 6911,
                      "end": 6922
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6911,
                    "end": 6922
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 6927,
                      "end": 6936
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6927,
                    "end": 6936
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6941,
                      "end": 6950
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6941,
                    "end": 6950
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6955,
                      "end": 6962
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6955,
                    "end": 6962
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6967,
                      "end": 6979
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6967,
                    "end": 6979
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 6984,
                      "end": 6992
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6984,
                    "end": 6992
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 6997,
                      "end": 7011
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
                            "start": 7022,
                            "end": 7024
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7022,
                          "end": 7024
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 7033,
                            "end": 7043
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7033,
                          "end": 7043
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 7052,
                            "end": 7062
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7052,
                          "end": 7062
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 7071,
                            "end": 7078
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7071,
                          "end": 7078
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 7087,
                            "end": 7098
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7087,
                          "end": 7098
                        }
                      }
                    ],
                    "loc": {
                      "start": 7012,
                      "end": 7104
                    }
                  },
                  "loc": {
                    "start": 6997,
                    "end": 7104
                  }
                }
              ],
              "loc": {
                "start": 6873,
                "end": 7106
              }
            },
            "loc": {
              "start": 6869,
              "end": 7106
            }
          }
        ],
        "loc": {
          "start": 6652,
          "end": 7108
        }
      },
      "loc": {
        "start": 6625,
        "end": 7108
      }
    },
    "Team_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_nav",
        "loc": {
          "start": 7118,
          "end": 7126
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 7130,
            "end": 7134
          }
        },
        "loc": {
          "start": 7130,
          "end": 7134
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
                "start": 7137,
                "end": 7139
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7137,
              "end": 7139
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7140,
                "end": 7151
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7140,
              "end": 7151
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7152,
                "end": 7158
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7152,
              "end": 7158
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7159,
                "end": 7171
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7159,
              "end": 7171
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7172,
                "end": 7175
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
                      "start": 7182,
                      "end": 7195
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7182,
                    "end": 7195
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 7200,
                      "end": 7209
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7200,
                    "end": 7209
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 7214,
                      "end": 7225
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7214,
                    "end": 7225
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 7230,
                      "end": 7239
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7230,
                    "end": 7239
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7244,
                      "end": 7253
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7244,
                    "end": 7253
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7258,
                      "end": 7265
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7258,
                    "end": 7265
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7270,
                      "end": 7282
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7270,
                    "end": 7282
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7287,
                      "end": 7295
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7287,
                    "end": 7295
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 7300,
                      "end": 7314
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
                            "start": 7325,
                            "end": 7327
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7325,
                          "end": 7327
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 7336,
                            "end": 7346
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7336,
                          "end": 7346
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 7355,
                            "end": 7365
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7355,
                          "end": 7365
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 7374,
                            "end": 7381
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7374,
                          "end": 7381
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 7390,
                            "end": 7401
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7390,
                          "end": 7401
                        }
                      }
                    ],
                    "loc": {
                      "start": 7315,
                      "end": 7407
                    }
                  },
                  "loc": {
                    "start": 7300,
                    "end": 7407
                  }
                }
              ],
              "loc": {
                "start": 7176,
                "end": 7409
              }
            },
            "loc": {
              "start": 7172,
              "end": 7409
            }
          }
        ],
        "loc": {
          "start": 7135,
          "end": 7411
        }
      },
      "loc": {
        "start": 7109,
        "end": 7411
      }
    },
    "User_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_list",
        "loc": {
          "start": 7421,
          "end": 7430
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7434,
            "end": 7438
          }
        },
        "loc": {
          "start": 7434,
          "end": 7438
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
                "start": 7441,
                "end": 7453
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
                      "start": 7460,
                      "end": 7462
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7460,
                    "end": 7462
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 7467,
                      "end": 7475
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7467,
                    "end": 7475
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 7480,
                      "end": 7483
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7480,
                    "end": 7483
                  }
                }
              ],
              "loc": {
                "start": 7454,
                "end": 7485
              }
            },
            "loc": {
              "start": 7441,
              "end": 7485
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7486,
                "end": 7488
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7486,
              "end": 7488
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7489,
                "end": 7499
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7489,
              "end": 7499
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7500,
                "end": 7510
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7500,
              "end": 7510
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7511,
                "end": 7522
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7511,
              "end": 7522
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7523,
                "end": 7529
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7523,
              "end": 7529
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7530,
                "end": 7535
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7530,
              "end": 7535
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 7536,
                "end": 7556
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7536,
              "end": 7556
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7557,
                "end": 7561
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7557,
              "end": 7561
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7562,
                "end": 7574
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7562,
              "end": 7574
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7575,
                "end": 7584
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7575,
              "end": 7584
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsReceivedCount",
              "loc": {
                "start": 7585,
                "end": 7605
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7585,
              "end": 7605
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7606,
                "end": 7609
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
                      "start": 7616,
                      "end": 7625
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7616,
                    "end": 7625
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 7630,
                      "end": 7639
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7630,
                    "end": 7639
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7644,
                      "end": 7653
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7644,
                    "end": 7653
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7658,
                      "end": 7670
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7658,
                    "end": 7670
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7675,
                      "end": 7683
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7675,
                    "end": 7683
                  }
                }
              ],
              "loc": {
                "start": 7610,
                "end": 7685
              }
            },
            "loc": {
              "start": 7606,
              "end": 7685
            }
          }
        ],
        "loc": {
          "start": 7439,
          "end": 7687
        }
      },
      "loc": {
        "start": 7412,
        "end": 7687
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 7697,
          "end": 7705
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7709,
            "end": 7713
          }
        },
        "loc": {
          "start": 7709,
          "end": 7713
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
                "start": 7716,
                "end": 7718
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7716,
              "end": 7718
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7719,
                "end": 7729
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7719,
              "end": 7729
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7730,
                "end": 7740
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7730,
              "end": 7740
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7741,
                "end": 7752
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7741,
              "end": 7752
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7753,
                "end": 7759
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7753,
              "end": 7759
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7760,
                "end": 7765
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7760,
              "end": 7765
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 7766,
                "end": 7786
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7766,
              "end": 7786
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7787,
                "end": 7791
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7787,
              "end": 7791
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7792,
                "end": 7804
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7792,
              "end": 7804
            }
          }
        ],
        "loc": {
          "start": 7714,
          "end": 7806
        }
      },
      "loc": {
        "start": 7688,
        "end": 7806
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
        "start": 7814,
        "end": 7822
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
              "start": 7824,
              "end": 7829
            }
          },
          "loc": {
            "start": 7823,
            "end": 7829
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
                "start": 7831,
                "end": 7849
              }
            },
            "loc": {
              "start": 7831,
              "end": 7849
            }
          },
          "loc": {
            "start": 7831,
            "end": 7850
          }
        },
        "directives": [],
        "loc": {
          "start": 7823,
          "end": 7850
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
              "start": 7856,
              "end": 7864
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 7865,
                  "end": 7870
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 7873,
                    "end": 7878
                  }
                },
                "loc": {
                  "start": 7872,
                  "end": 7878
                }
              },
              "loc": {
                "start": 7865,
                "end": 7878
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
                    "start": 7886,
                    "end": 7891
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
                          "start": 7902,
                          "end": 7908
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 7902,
                        "end": 7908
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 7917,
                          "end": 7921
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
                                  "start": 7943,
                                  "end": 7946
                                }
                              },
                              "loc": {
                                "start": 7943,
                                "end": 7946
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
                                      "start": 7968,
                                      "end": 7976
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 7965,
                                    "end": 7976
                                  }
                                }
                              ],
                              "loc": {
                                "start": 7947,
                                "end": 7990
                              }
                            },
                            "loc": {
                              "start": 7936,
                              "end": 7990
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
                                  "start": 8010,
                                  "end": 8014
                                }
                              },
                              "loc": {
                                "start": 8010,
                                "end": 8014
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
                                      "start": 8036,
                                      "end": 8045
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8033,
                                    "end": 8045
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8015,
                                "end": 8059
                              }
                            },
                            "loc": {
                              "start": 8003,
                              "end": 8059
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
                                  "start": 8079,
                                  "end": 8083
                                }
                              },
                              "loc": {
                                "start": 8079,
                                "end": 8083
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
                                      "start": 8105,
                                      "end": 8114
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8102,
                                    "end": 8114
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8084,
                                "end": 8128
                              }
                            },
                            "loc": {
                              "start": 8072,
                              "end": 8128
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
                                  "start": 8148,
                                  "end": 8155
                                }
                              },
                              "loc": {
                                "start": 8148,
                                "end": 8155
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
                                      "start": 8177,
                                      "end": 8189
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8174,
                                    "end": 8189
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8156,
                                "end": 8203
                              }
                            },
                            "loc": {
                              "start": 8141,
                              "end": 8203
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
                                  "start": 8223,
                                  "end": 8231
                                }
                              },
                              "loc": {
                                "start": 8223,
                                "end": 8231
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
                                      "start": 8253,
                                      "end": 8266
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8250,
                                    "end": 8266
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8232,
                                "end": 8280
                              }
                            },
                            "loc": {
                              "start": 8216,
                              "end": 8280
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
                                  "start": 8300,
                                  "end": 8307
                                }
                              },
                              "loc": {
                                "start": 8300,
                                "end": 8307
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
                                      "start": 8329,
                                      "end": 8341
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8326,
                                    "end": 8341
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8308,
                                "end": 8355
                              }
                            },
                            "loc": {
                              "start": 8293,
                              "end": 8355
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
                                  "start": 8375,
                                  "end": 8383
                                }
                              },
                              "loc": {
                                "start": 8375,
                                "end": 8383
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
                                      "start": 8405,
                                      "end": 8418
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8402,
                                    "end": 8418
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8384,
                                "end": 8432
                              }
                            },
                            "loc": {
                              "start": 8368,
                              "end": 8432
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
                                  "start": 8452,
                                  "end": 8456
                                }
                              },
                              "loc": {
                                "start": 8452,
                                "end": 8456
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
                                      "start": 8478,
                                      "end": 8487
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8475,
                                    "end": 8487
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8457,
                                "end": 8501
                              }
                            },
                            "loc": {
                              "start": 8445,
                              "end": 8501
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
                                  "start": 8521,
                                  "end": 8525
                                }
                              },
                              "loc": {
                                "start": 8521,
                                "end": 8525
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
                                      "start": 8547,
                                      "end": 8556
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8544,
                                    "end": 8556
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8526,
                                "end": 8570
                              }
                            },
                            "loc": {
                              "start": 8514,
                              "end": 8570
                            }
                          }
                        ],
                        "loc": {
                          "start": 7922,
                          "end": 8580
                        }
                      },
                      "loc": {
                        "start": 7917,
                        "end": 8580
                      }
                    }
                  ],
                  "loc": {
                    "start": 7892,
                    "end": 8586
                  }
                },
                "loc": {
                  "start": 7886,
                  "end": 8586
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 8591,
                    "end": 8599
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
                          "start": 8610,
                          "end": 8621
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8610,
                        "end": 8621
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorApi",
                        "loc": {
                          "start": 8630,
                          "end": 8642
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8630,
                        "end": 8642
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorCode",
                        "loc": {
                          "start": 8651,
                          "end": 8664
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8651,
                        "end": 8664
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorNote",
                        "loc": {
                          "start": 8673,
                          "end": 8686
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8673,
                        "end": 8686
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 8695,
                          "end": 8711
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8695,
                        "end": 8711
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorQuestion",
                        "loc": {
                          "start": 8720,
                          "end": 8737
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8720,
                        "end": 8737
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRoutine",
                        "loc": {
                          "start": 8746,
                          "end": 8762
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8746,
                        "end": 8762
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorStandard",
                        "loc": {
                          "start": 8771,
                          "end": 8788
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8771,
                        "end": 8788
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorTeam",
                        "loc": {
                          "start": 8797,
                          "end": 8810
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8797,
                        "end": 8810
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorUser",
                        "loc": {
                          "start": 8819,
                          "end": 8832
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8819,
                        "end": 8832
                      }
                    }
                  ],
                  "loc": {
                    "start": 8600,
                    "end": 8838
                  }
                },
                "loc": {
                  "start": 8591,
                  "end": 8838
                }
              }
            ],
            "loc": {
              "start": 7880,
              "end": 8842
            }
          },
          "loc": {
            "start": 7856,
            "end": 8842
          }
        }
      ],
      "loc": {
        "start": 7852,
        "end": 8844
      }
    },
    "loc": {
      "start": 7808,
      "end": 8844
    }
  },
  "variableValues": {},
  "path": {
    "key": "popular_findMany"
  }
} as const;
