export const popular_findMany = {
  "fieldName": "populars",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "populars",
        "loc": {
          "start": 8012,
          "end": 8020
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 8021,
              "end": 8026
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 8029,
                "end": 8034
              }
            },
            "loc": {
              "start": 8028,
              "end": 8034
            }
          },
          "loc": {
            "start": 8021,
            "end": 8034
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
                "start": 8042,
                "end": 8047
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
                      "start": 8058,
                      "end": 8064
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8058,
                    "end": 8064
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 8073,
                      "end": 8077
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
                              "start": 8099,
                              "end": 8102
                            }
                          },
                          "loc": {
                            "start": 8099,
                            "end": 8102
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
                                  "start": 8124,
                                  "end": 8132
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8121,
                                "end": 8132
                              }
                            }
                          ],
                          "loc": {
                            "start": 8103,
                            "end": 8146
                          }
                        },
                        "loc": {
                          "start": 8092,
                          "end": 8146
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
                              "start": 8166,
                              "end": 8170
                            }
                          },
                          "loc": {
                            "start": 8166,
                            "end": 8170
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
                                  "start": 8192,
                                  "end": 8201
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8189,
                                "end": 8201
                              }
                            }
                          ],
                          "loc": {
                            "start": 8171,
                            "end": 8215
                          }
                        },
                        "loc": {
                          "start": 8159,
                          "end": 8215
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Organization",
                            "loc": {
                              "start": 8235,
                              "end": 8247
                            }
                          },
                          "loc": {
                            "start": 8235,
                            "end": 8247
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
                                "value": "Organization_list",
                                "loc": {
                                  "start": 8269,
                                  "end": 8286
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8266,
                                "end": 8286
                              }
                            }
                          ],
                          "loc": {
                            "start": 8248,
                            "end": 8300
                          }
                        },
                        "loc": {
                          "start": 8228,
                          "end": 8300
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
                              "start": 8320,
                              "end": 8327
                            }
                          },
                          "loc": {
                            "start": 8320,
                            "end": 8327
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
                                  "start": 8349,
                                  "end": 8361
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8346,
                                "end": 8361
                              }
                            }
                          ],
                          "loc": {
                            "start": 8328,
                            "end": 8375
                          }
                        },
                        "loc": {
                          "start": 8313,
                          "end": 8375
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
                              "start": 8395,
                              "end": 8403
                            }
                          },
                          "loc": {
                            "start": 8395,
                            "end": 8403
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
                                  "start": 8425,
                                  "end": 8438
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8422,
                                "end": 8438
                              }
                            }
                          ],
                          "loc": {
                            "start": 8404,
                            "end": 8452
                          }
                        },
                        "loc": {
                          "start": 8388,
                          "end": 8452
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
                              "start": 8472,
                              "end": 8479
                            }
                          },
                          "loc": {
                            "start": 8472,
                            "end": 8479
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
                                  "start": 8501,
                                  "end": 8513
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8498,
                                "end": 8513
                              }
                            }
                          ],
                          "loc": {
                            "start": 8480,
                            "end": 8527
                          }
                        },
                        "loc": {
                          "start": 8465,
                          "end": 8527
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "SmartContract",
                            "loc": {
                              "start": 8547,
                              "end": 8560
                            }
                          },
                          "loc": {
                            "start": 8547,
                            "end": 8560
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
                                "value": "SmartContract_list",
                                "loc": {
                                  "start": 8582,
                                  "end": 8600
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8579,
                                "end": 8600
                              }
                            }
                          ],
                          "loc": {
                            "start": 8561,
                            "end": 8614
                          }
                        },
                        "loc": {
                          "start": 8540,
                          "end": 8614
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
                              "start": 8634,
                              "end": 8642
                            }
                          },
                          "loc": {
                            "start": 8634,
                            "end": 8642
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
                                  "start": 8664,
                                  "end": 8677
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8661,
                                "end": 8677
                              }
                            }
                          ],
                          "loc": {
                            "start": 8643,
                            "end": 8691
                          }
                        },
                        "loc": {
                          "start": 8627,
                          "end": 8691
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
                              "start": 8711,
                              "end": 8715
                            }
                          },
                          "loc": {
                            "start": 8711,
                            "end": 8715
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
                                  "start": 8737,
                                  "end": 8746
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8734,
                                "end": 8746
                              }
                            }
                          ],
                          "loc": {
                            "start": 8716,
                            "end": 8760
                          }
                        },
                        "loc": {
                          "start": 8704,
                          "end": 8760
                        }
                      }
                    ],
                    "loc": {
                      "start": 8078,
                      "end": 8770
                    }
                  },
                  "loc": {
                    "start": 8073,
                    "end": 8770
                  }
                }
              ],
              "loc": {
                "start": 8048,
                "end": 8776
              }
            },
            "loc": {
              "start": 8042,
              "end": 8776
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 8781,
                "end": 8789
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
                      "start": 8800,
                      "end": 8811
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8800,
                    "end": 8811
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorApi",
                    "loc": {
                      "start": 8820,
                      "end": 8832
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8820,
                    "end": 8832
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorNote",
                    "loc": {
                      "start": 8841,
                      "end": 8854
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8841,
                    "end": 8854
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorOrganization",
                    "loc": {
                      "start": 8863,
                      "end": 8884
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8863,
                    "end": 8884
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 8893,
                      "end": 8909
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8893,
                    "end": 8909
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorQuestion",
                    "loc": {
                      "start": 8918,
                      "end": 8935
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8918,
                    "end": 8935
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRoutine",
                    "loc": {
                      "start": 8944,
                      "end": 8960
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8944,
                    "end": 8960
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorSmartContract",
                    "loc": {
                      "start": 8969,
                      "end": 8991
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8969,
                    "end": 8991
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorStandard",
                    "loc": {
                      "start": 9000,
                      "end": 9017
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9000,
                    "end": 9017
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorUser",
                    "loc": {
                      "start": 9026,
                      "end": 9039
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9026,
                    "end": 9039
                  }
                }
              ],
              "loc": {
                "start": 8790,
                "end": 9045
              }
            },
            "loc": {
              "start": 8781,
              "end": 9045
            }
          }
        ],
        "loc": {
          "start": 8036,
          "end": 9049
        }
      },
      "loc": {
        "start": 8012,
        "end": 9049
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
                "value": "Organization",
                "loc": {
                  "start": 553,
                  "end": 565
                }
              },
              "loc": {
                "start": 553,
                "end": 565
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 579,
                      "end": 595
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 576,
                    "end": 595
                  }
                }
              ],
              "loc": {
                "start": 566,
                "end": 601
              }
            },
            "loc": {
              "start": 546,
              "end": 601
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
                  "start": 613,
                  "end": 617
                }
              },
              "loc": {
                "start": 613,
                "end": 617
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
                      "start": 631,
                      "end": 639
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 628,
                    "end": 639
                  }
                }
              ],
              "loc": {
                "start": 618,
                "end": 645
              }
            },
            "loc": {
              "start": 606,
              "end": 645
            }
          }
        ],
        "loc": {
          "start": 540,
          "end": 647
        }
      },
      "loc": {
        "start": 534,
        "end": 647
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 648,
          "end": 659
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 648,
        "end": 659
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 660,
          "end": 674
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 660,
        "end": 674
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 675,
          "end": 680
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 675,
        "end": 680
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 681,
          "end": 690
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 681,
        "end": 690
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 691,
          "end": 695
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
                "start": 705,
                "end": 713
              }
            },
            "directives": [],
            "loc": {
              "start": 702,
              "end": 713
            }
          }
        ],
        "loc": {
          "start": 696,
          "end": 715
        }
      },
      "loc": {
        "start": 691,
        "end": 715
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 716,
          "end": 730
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 716,
        "end": 730
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 731,
          "end": 736
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 731,
        "end": 736
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 737,
          "end": 740
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
                "start": 747,
                "end": 756
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 747,
              "end": 756
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
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
              "value": "canTransfer",
              "loc": {
                "start": 777,
                "end": 788
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 777,
              "end": 788
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 793,
                "end": 802
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 793,
              "end": 802
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 807,
                "end": 814
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 807,
              "end": 814
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 819,
                "end": 827
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 819,
              "end": 827
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 832,
                "end": 844
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 832,
              "end": 844
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 849,
                "end": 857
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 849,
              "end": 857
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 862,
                "end": 870
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 862,
              "end": 870
            }
          }
        ],
        "loc": {
          "start": 741,
          "end": 872
        }
      },
      "loc": {
        "start": 737,
        "end": 872
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 901,
          "end": 903
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 901,
        "end": 903
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 904,
          "end": 913
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 904,
        "end": 913
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 947,
          "end": 949
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 947,
        "end": 949
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 950,
          "end": 960
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 950,
        "end": 960
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 961,
          "end": 971
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 961,
        "end": 971
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 972,
          "end": 977
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 972,
        "end": 977
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 978,
          "end": 983
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 978,
        "end": 983
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 984,
          "end": 989
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
                "value": "Organization",
                "loc": {
                  "start": 1003,
                  "end": 1015
                }
              },
              "loc": {
                "start": 1003,
                "end": 1015
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 1029,
                      "end": 1045
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1026,
                    "end": 1045
                  }
                }
              ],
              "loc": {
                "start": 1016,
                "end": 1051
              }
            },
            "loc": {
              "start": 996,
              "end": 1051
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
                  "start": 1063,
                  "end": 1067
                }
              },
              "loc": {
                "start": 1063,
                "end": 1067
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
                      "start": 1081,
                      "end": 1089
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1078,
                    "end": 1089
                  }
                }
              ],
              "loc": {
                "start": 1068,
                "end": 1095
              }
            },
            "loc": {
              "start": 1056,
              "end": 1095
            }
          }
        ],
        "loc": {
          "start": 990,
          "end": 1097
        }
      },
      "loc": {
        "start": 984,
        "end": 1097
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1098,
          "end": 1101
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
                "start": 1108,
                "end": 1117
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1108,
              "end": 1117
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1122,
                "end": 1131
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1122,
              "end": 1131
            }
          }
        ],
        "loc": {
          "start": 1102,
          "end": 1133
        }
      },
      "loc": {
        "start": 1098,
        "end": 1133
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 1165,
          "end": 1173
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
                "start": 1180,
                "end": 1192
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
                      "start": 1203,
                      "end": 1205
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1203,
                    "end": 1205
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1214,
                      "end": 1222
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1214,
                    "end": 1222
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1231,
                      "end": 1242
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1231,
                    "end": 1242
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1251,
                      "end": 1255
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1251,
                    "end": 1255
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "pages",
                    "loc": {
                      "start": 1264,
                      "end": 1269
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
                            "start": 1284,
                            "end": 1286
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1284,
                          "end": 1286
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pageIndex",
                          "loc": {
                            "start": 1299,
                            "end": 1308
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1299,
                          "end": 1308
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 1321,
                            "end": 1325
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1321,
                          "end": 1325
                        }
                      }
                    ],
                    "loc": {
                      "start": 1270,
                      "end": 1335
                    }
                  },
                  "loc": {
                    "start": 1264,
                    "end": 1335
                  }
                }
              ],
              "loc": {
                "start": 1193,
                "end": 1341
              }
            },
            "loc": {
              "start": 1180,
              "end": 1341
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1346,
                "end": 1348
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1346,
              "end": 1348
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1353,
                "end": 1363
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1353,
              "end": 1363
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1368,
                "end": 1378
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1368,
              "end": 1378
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1383,
                "end": 1391
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1383,
              "end": 1391
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1396,
                "end": 1405
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1396,
              "end": 1405
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1410,
                "end": 1422
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1410,
              "end": 1422
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1427,
                "end": 1439
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1427,
              "end": 1439
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1444,
                "end": 1456
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1444,
              "end": 1456
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1461,
                "end": 1464
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
                      "start": 1475,
                      "end": 1485
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1475,
                    "end": 1485
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 1494,
                      "end": 1501
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1494,
                    "end": 1501
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1510,
                      "end": 1519
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1510,
                    "end": 1519
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1528,
                      "end": 1537
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1528,
                    "end": 1537
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1546,
                      "end": 1555
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1546,
                    "end": 1555
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 1564,
                      "end": 1570
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1564,
                    "end": 1570
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1579,
                      "end": 1586
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1579,
                    "end": 1586
                  }
                }
              ],
              "loc": {
                "start": 1465,
                "end": 1592
              }
            },
            "loc": {
              "start": 1461,
              "end": 1592
            }
          }
        ],
        "loc": {
          "start": 1174,
          "end": 1594
        }
      },
      "loc": {
        "start": 1165,
        "end": 1594
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1595,
          "end": 1597
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1595,
        "end": 1597
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1598,
          "end": 1608
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1598,
        "end": 1608
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1609,
          "end": 1619
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1609,
        "end": 1619
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1620,
          "end": 1629
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1620,
        "end": 1629
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 1630,
          "end": 1641
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1630,
        "end": 1641
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1642,
          "end": 1648
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
                "start": 1658,
                "end": 1668
              }
            },
            "directives": [],
            "loc": {
              "start": 1655,
              "end": 1668
            }
          }
        ],
        "loc": {
          "start": 1649,
          "end": 1670
        }
      },
      "loc": {
        "start": 1642,
        "end": 1670
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1671,
          "end": 1676
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
                "value": "Organization",
                "loc": {
                  "start": 1690,
                  "end": 1702
                }
              },
              "loc": {
                "start": 1690,
                "end": 1702
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 1716,
                      "end": 1732
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1713,
                    "end": 1732
                  }
                }
              ],
              "loc": {
                "start": 1703,
                "end": 1738
              }
            },
            "loc": {
              "start": 1683,
              "end": 1738
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
                  "start": 1750,
                  "end": 1754
                }
              },
              "loc": {
                "start": 1750,
                "end": 1754
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
                      "start": 1768,
                      "end": 1776
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1765,
                    "end": 1776
                  }
                }
              ],
              "loc": {
                "start": 1755,
                "end": 1782
              }
            },
            "loc": {
              "start": 1743,
              "end": 1782
            }
          }
        ],
        "loc": {
          "start": 1677,
          "end": 1784
        }
      },
      "loc": {
        "start": 1671,
        "end": 1784
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1785,
          "end": 1796
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1785,
        "end": 1796
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 1797,
          "end": 1811
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1797,
        "end": 1811
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 1812,
          "end": 1817
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1812,
        "end": 1817
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1818,
          "end": 1827
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1818,
        "end": 1827
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1828,
          "end": 1832
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
                "start": 1842,
                "end": 1850
              }
            },
            "directives": [],
            "loc": {
              "start": 1839,
              "end": 1850
            }
          }
        ],
        "loc": {
          "start": 1833,
          "end": 1852
        }
      },
      "loc": {
        "start": 1828,
        "end": 1852
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 1853,
          "end": 1867
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1853,
        "end": 1867
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 1868,
          "end": 1873
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1868,
        "end": 1873
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1874,
          "end": 1877
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
                "start": 1884,
                "end": 1893
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1884,
              "end": 1893
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1898,
                "end": 1909
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1898,
              "end": 1909
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 1914,
                "end": 1925
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1914,
              "end": 1925
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1930,
                "end": 1939
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1930,
              "end": 1939
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1944,
                "end": 1951
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1944,
              "end": 1951
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1956,
                "end": 1964
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1956,
              "end": 1964
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1969,
                "end": 1981
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1969,
              "end": 1981
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1986,
                "end": 1994
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1986,
              "end": 1994
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1999,
                "end": 2007
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1999,
              "end": 2007
            }
          }
        ],
        "loc": {
          "start": 1878,
          "end": 2009
        }
      },
      "loc": {
        "start": 1874,
        "end": 2009
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2040,
          "end": 2042
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2040,
        "end": 2042
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2043,
          "end": 2052
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2043,
        "end": 2052
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2100,
          "end": 2102
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2100,
        "end": 2102
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 2103,
          "end": 2114
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2103,
        "end": 2114
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2115,
          "end": 2121
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2115,
        "end": 2121
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2122,
          "end": 2132
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2122,
        "end": 2132
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2133,
          "end": 2143
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2133,
        "end": 2143
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isOpenToNewMembers",
        "loc": {
          "start": 2144,
          "end": 2162
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2144,
        "end": 2162
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2163,
          "end": 2172
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2163,
        "end": 2172
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 2173,
          "end": 2186
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2173,
        "end": 2186
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "membersCount",
        "loc": {
          "start": 2187,
          "end": 2199
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2187,
        "end": 2199
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 2200,
          "end": 2212
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2200,
        "end": 2212
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 2213,
          "end": 2225
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2213,
        "end": 2225
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2226,
          "end": 2235
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2226,
        "end": 2235
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 2236,
          "end": 2240
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
                "start": 2250,
                "end": 2258
              }
            },
            "directives": [],
            "loc": {
              "start": 2247,
              "end": 2258
            }
          }
        ],
        "loc": {
          "start": 2241,
          "end": 2260
        }
      },
      "loc": {
        "start": 2236,
        "end": 2260
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 2261,
          "end": 2273
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
                "start": 2280,
                "end": 2282
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2280,
              "end": 2282
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 2287,
                "end": 2295
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2287,
              "end": 2295
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 2300,
                "end": 2303
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2300,
              "end": 2303
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2308,
                "end": 2312
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2308,
              "end": 2312
            }
          }
        ],
        "loc": {
          "start": 2274,
          "end": 2314
        }
      },
      "loc": {
        "start": 2261,
        "end": 2314
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2315,
          "end": 2318
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
                "start": 2325,
                "end": 2338
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2325,
              "end": 2338
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2343,
                "end": 2352
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2343,
              "end": 2352
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2357,
                "end": 2368
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2357,
              "end": 2368
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 2373,
                "end": 2382
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2373,
              "end": 2382
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2387,
                "end": 2396
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2387,
              "end": 2396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2401,
                "end": 2408
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2401,
              "end": 2408
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2413,
                "end": 2425
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2413,
              "end": 2425
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2430,
                "end": 2438
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2430,
              "end": 2438
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 2443,
                "end": 2457
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
                      "start": 2468,
                      "end": 2470
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2468,
                    "end": 2470
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2479,
                      "end": 2489
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2479,
                    "end": 2489
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2498,
                      "end": 2508
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2498,
                    "end": 2508
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 2517,
                      "end": 2524
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2517,
                    "end": 2524
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 2533,
                      "end": 2544
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2533,
                    "end": 2544
                  }
                }
              ],
              "loc": {
                "start": 2458,
                "end": 2550
              }
            },
            "loc": {
              "start": 2443,
              "end": 2550
            }
          }
        ],
        "loc": {
          "start": 2319,
          "end": 2552
        }
      },
      "loc": {
        "start": 2315,
        "end": 2552
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2599,
          "end": 2601
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2599,
        "end": 2601
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 2602,
          "end": 2613
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2602,
        "end": 2613
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2614,
          "end": 2620
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2614,
        "end": 2620
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 2621,
          "end": 2633
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2621,
        "end": 2633
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2634,
          "end": 2637
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
                "start": 2644,
                "end": 2657
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2644,
              "end": 2657
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2662,
                "end": 2671
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2662,
              "end": 2671
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2676,
                "end": 2687
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2676,
              "end": 2687
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 2692,
                "end": 2701
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2692,
              "end": 2701
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2706,
                "end": 2715
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2706,
              "end": 2715
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2720,
                "end": 2727
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2720,
              "end": 2727
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2732,
                "end": 2744
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2732,
              "end": 2744
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2749,
                "end": 2757
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2749,
              "end": 2757
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 2762,
                "end": 2776
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
                      "start": 2787,
                      "end": 2789
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2787,
                    "end": 2789
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2798,
                      "end": 2808
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2798,
                    "end": 2808
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2817,
                      "end": 2827
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2817,
                    "end": 2827
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 2836,
                      "end": 2843
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2836,
                    "end": 2843
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 2852,
                      "end": 2863
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2852,
                    "end": 2863
                  }
                }
              ],
              "loc": {
                "start": 2777,
                "end": 2869
              }
            },
            "loc": {
              "start": 2762,
              "end": 2869
            }
          }
        ],
        "loc": {
          "start": 2638,
          "end": 2871
        }
      },
      "loc": {
        "start": 2634,
        "end": 2871
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 2909,
          "end": 2917
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
                "start": 2924,
                "end": 2936
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
                      "start": 2947,
                      "end": 2949
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2947,
                    "end": 2949
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2958,
                      "end": 2966
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2958,
                    "end": 2966
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2975,
                      "end": 2986
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2975,
                    "end": 2986
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2995,
                      "end": 2999
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2995,
                    "end": 2999
                  }
                }
              ],
              "loc": {
                "start": 2937,
                "end": 3005
              }
            },
            "loc": {
              "start": 2924,
              "end": 3005
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3010,
                "end": 3012
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3010,
              "end": 3012
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3017,
                "end": 3027
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3017,
              "end": 3027
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3032,
                "end": 3042
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3032,
              "end": 3042
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 3047,
                "end": 3063
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3047,
              "end": 3063
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 3068,
                "end": 3076
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3068,
              "end": 3076
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3081,
                "end": 3090
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3081,
              "end": 3090
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 3095,
                "end": 3107
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3095,
              "end": 3107
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 3112,
                "end": 3128
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3112,
              "end": 3128
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 3133,
                "end": 3143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3133,
              "end": 3143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 3148,
                "end": 3160
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3148,
              "end": 3160
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 3165,
                "end": 3177
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3165,
              "end": 3177
            }
          }
        ],
        "loc": {
          "start": 2918,
          "end": 3179
        }
      },
      "loc": {
        "start": 2909,
        "end": 3179
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3180,
          "end": 3182
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3180,
        "end": 3182
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3183,
          "end": 3193
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3183,
        "end": 3193
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3194,
          "end": 3204
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3194,
        "end": 3204
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3205,
          "end": 3214
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3205,
        "end": 3214
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 3215,
          "end": 3226
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3215,
        "end": 3226
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 3227,
          "end": 3233
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
                "start": 3243,
                "end": 3253
              }
            },
            "directives": [],
            "loc": {
              "start": 3240,
              "end": 3253
            }
          }
        ],
        "loc": {
          "start": 3234,
          "end": 3255
        }
      },
      "loc": {
        "start": 3227,
        "end": 3255
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 3256,
          "end": 3261
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
                "value": "Organization",
                "loc": {
                  "start": 3275,
                  "end": 3287
                }
              },
              "loc": {
                "start": 3275,
                "end": 3287
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 3301,
                      "end": 3317
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3298,
                    "end": 3317
                  }
                }
              ],
              "loc": {
                "start": 3288,
                "end": 3323
              }
            },
            "loc": {
              "start": 3268,
              "end": 3323
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
                  "start": 3335,
                  "end": 3339
                }
              },
              "loc": {
                "start": 3335,
                "end": 3339
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
                      "start": 3353,
                      "end": 3361
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3350,
                    "end": 3361
                  }
                }
              ],
              "loc": {
                "start": 3340,
                "end": 3367
              }
            },
            "loc": {
              "start": 3328,
              "end": 3367
            }
          }
        ],
        "loc": {
          "start": 3262,
          "end": 3369
        }
      },
      "loc": {
        "start": 3256,
        "end": 3369
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 3370,
          "end": 3381
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3370,
        "end": 3381
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 3382,
          "end": 3396
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3382,
        "end": 3396
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 3397,
          "end": 3402
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3397,
        "end": 3402
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 3403,
          "end": 3412
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3403,
        "end": 3412
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 3413,
          "end": 3417
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
                "start": 3427,
                "end": 3435
              }
            },
            "directives": [],
            "loc": {
              "start": 3424,
              "end": 3435
            }
          }
        ],
        "loc": {
          "start": 3418,
          "end": 3437
        }
      },
      "loc": {
        "start": 3413,
        "end": 3437
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 3438,
          "end": 3452
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3438,
        "end": 3452
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 3453,
          "end": 3458
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3453,
        "end": 3458
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 3459,
          "end": 3462
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
                "start": 3469,
                "end": 3478
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3469,
              "end": 3478
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 3483,
                "end": 3494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3483,
              "end": 3494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 3499,
                "end": 3510
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3499,
              "end": 3510
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 3515,
                "end": 3524
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3515,
              "end": 3524
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 3529,
                "end": 3536
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3529,
              "end": 3536
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 3541,
                "end": 3549
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3541,
              "end": 3549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 3554,
                "end": 3566
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3554,
              "end": 3566
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 3571,
                "end": 3579
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3571,
              "end": 3579
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 3584,
                "end": 3592
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3584,
              "end": 3592
            }
          }
        ],
        "loc": {
          "start": 3463,
          "end": 3594
        }
      },
      "loc": {
        "start": 3459,
        "end": 3594
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3631,
          "end": 3633
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3631,
        "end": 3633
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3634,
          "end": 3643
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3634,
        "end": 3643
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 3683,
          "end": 3695
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
                "start": 3702,
                "end": 3704
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3702,
              "end": 3704
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 3709,
                "end": 3717
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3709,
              "end": 3717
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 3722,
                "end": 3733
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3722,
              "end": 3733
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3738,
                "end": 3742
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3738,
              "end": 3742
            }
          }
        ],
        "loc": {
          "start": 3696,
          "end": 3744
        }
      },
      "loc": {
        "start": 3683,
        "end": 3744
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3745,
          "end": 3747
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3745,
        "end": 3747
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3748,
          "end": 3758
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3748,
        "end": 3758
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3759,
          "end": 3769
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3759,
        "end": 3769
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "createdBy",
        "loc": {
          "start": 3770,
          "end": 3779
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
                "start": 3786,
                "end": 3788
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3786,
              "end": 3788
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3793,
                "end": 3803
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3793,
              "end": 3803
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3808,
                "end": 3818
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3808,
              "end": 3818
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 3823,
                "end": 3834
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3823,
              "end": 3834
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 3839,
                "end": 3845
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3839,
              "end": 3845
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 3850,
                "end": 3855
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3850,
              "end": 3855
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3860,
                "end": 3864
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3860,
              "end": 3864
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 3869,
                "end": 3881
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3869,
              "end": 3881
            }
          }
        ],
        "loc": {
          "start": 3780,
          "end": 3883
        }
      },
      "loc": {
        "start": 3770,
        "end": 3883
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "hasAcceptedAnswer",
        "loc": {
          "start": 3884,
          "end": 3901
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3884,
        "end": 3901
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3902,
          "end": 3911
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3902,
        "end": 3911
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 3912,
          "end": 3917
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3912,
        "end": 3917
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 3918,
          "end": 3927
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3918,
        "end": 3927
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "answersCount",
        "loc": {
          "start": 3928,
          "end": 3940
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3928,
        "end": 3940
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 3941,
          "end": 3954
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3941,
        "end": 3954
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 3955,
          "end": 3967
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3955,
        "end": 3967
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "forObject",
        "loc": {
          "start": 3968,
          "end": 3977
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
                  "start": 3991,
                  "end": 3994
                }
              },
              "loc": {
                "start": 3991,
                "end": 3994
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
                      "start": 4008,
                      "end": 4015
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4005,
                    "end": 4015
                  }
                }
              ],
              "loc": {
                "start": 3995,
                "end": 4021
              }
            },
            "loc": {
              "start": 3984,
              "end": 4021
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
                  "start": 4033,
                  "end": 4037
                }
              },
              "loc": {
                "start": 4033,
                "end": 4037
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
                      "start": 4051,
                      "end": 4059
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4048,
                    "end": 4059
                  }
                }
              ],
              "loc": {
                "start": 4038,
                "end": 4065
              }
            },
            "loc": {
              "start": 4026,
              "end": 4065
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 4077,
                  "end": 4089
                }
              },
              "loc": {
                "start": 4077,
                "end": 4089
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 4103,
                      "end": 4119
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4100,
                    "end": 4119
                  }
                }
              ],
              "loc": {
                "start": 4090,
                "end": 4125
              }
            },
            "loc": {
              "start": 4070,
              "end": 4125
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
                  "start": 4137,
                  "end": 4144
                }
              },
              "loc": {
                "start": 4137,
                "end": 4144
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
                      "start": 4158,
                      "end": 4169
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4155,
                    "end": 4169
                  }
                }
              ],
              "loc": {
                "start": 4145,
                "end": 4175
              }
            },
            "loc": {
              "start": 4130,
              "end": 4175
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
                  "start": 4187,
                  "end": 4194
                }
              },
              "loc": {
                "start": 4187,
                "end": 4194
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
                      "start": 4208,
                      "end": 4219
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4205,
                    "end": 4219
                  }
                }
              ],
              "loc": {
                "start": 4195,
                "end": 4225
              }
            },
            "loc": {
              "start": 4180,
              "end": 4225
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "SmartContract",
                "loc": {
                  "start": 4237,
                  "end": 4250
                }
              },
              "loc": {
                "start": 4237,
                "end": 4250
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
                    "value": "SmartContract_nav",
                    "loc": {
                      "start": 4264,
                      "end": 4281
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4261,
                    "end": 4281
                  }
                }
              ],
              "loc": {
                "start": 4251,
                "end": 4287
              }
            },
            "loc": {
              "start": 4230,
              "end": 4287
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
                  "start": 4299,
                  "end": 4307
                }
              },
              "loc": {
                "start": 4299,
                "end": 4307
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
                      "start": 4321,
                      "end": 4333
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4318,
                    "end": 4333
                  }
                }
              ],
              "loc": {
                "start": 4308,
                "end": 4339
              }
            },
            "loc": {
              "start": 4292,
              "end": 4339
            }
          }
        ],
        "loc": {
          "start": 3978,
          "end": 4341
        }
      },
      "loc": {
        "start": 3968,
        "end": 4341
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 4342,
          "end": 4346
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
                "start": 4356,
                "end": 4364
              }
            },
            "directives": [],
            "loc": {
              "start": 4353,
              "end": 4364
            }
          }
        ],
        "loc": {
          "start": 4347,
          "end": 4366
        }
      },
      "loc": {
        "start": 4342,
        "end": 4366
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 4367,
          "end": 4370
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
                "start": 4377,
                "end": 4385
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4377,
              "end": 4385
            }
          }
        ],
        "loc": {
          "start": 4371,
          "end": 4387
        }
      },
      "loc": {
        "start": 4367,
        "end": 4387
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 4425,
          "end": 4433
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
                "start": 4440,
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
                    "value": "id",
                    "loc": {
                      "start": 4463,
                      "end": 4465
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4463,
                    "end": 4465
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 4474,
                      "end": 4482
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4474,
                    "end": 4482
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 4491,
                      "end": 4502
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4491,
                    "end": 4502
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 4511,
                      "end": 4523
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4511,
                    "end": 4523
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4532,
                      "end": 4536
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4532,
                    "end": 4536
                  }
                }
              ],
              "loc": {
                "start": 4453,
                "end": 4542
              }
            },
            "loc": {
              "start": 4440,
              "end": 4542
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 4547,
                "end": 4549
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4547,
              "end": 4549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4554,
                "end": 4564
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4554,
              "end": 4564
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4569,
                "end": 4579
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4569,
              "end": 4579
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 4584,
                "end": 4595
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4584,
              "end": 4595
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 4600,
                "end": 4613
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4600,
              "end": 4613
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 4618,
                "end": 4628
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4618,
              "end": 4628
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 4633,
                "end": 4642
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4633,
              "end": 4642
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 4647,
                "end": 4655
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4647,
              "end": 4655
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4660,
                "end": 4669
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4660,
              "end": 4669
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 4674,
                "end": 4684
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4674,
              "end": 4684
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesStarted",
              "loc": {
                "start": 4689,
                "end": 4701
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4689,
              "end": 4701
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesCompleted",
              "loc": {
                "start": 4706,
                "end": 4720
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4706,
              "end": 4720
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContractCallData",
              "loc": {
                "start": 4725,
                "end": 4746
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4725,
              "end": 4746
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "apiCallData",
              "loc": {
                "start": 4751,
                "end": 4762
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4751,
              "end": 4762
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 4767,
                "end": 4779
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4767,
              "end": 4779
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 4784,
                "end": 4796
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4784,
              "end": 4796
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 4801,
                "end": 4814
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4801,
              "end": 4814
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 4819,
                "end": 4841
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4819,
              "end": 4841
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 4846,
                "end": 4856
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4846,
              "end": 4856
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 4861,
                "end": 4872
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4861,
              "end": 4872
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodesCount",
              "loc": {
                "start": 4877,
                "end": 4887
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4877,
              "end": 4887
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodeLinksCount",
              "loc": {
                "start": 4892,
                "end": 4906
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4892,
              "end": 4906
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
              "loc": {
                "start": 4911,
                "end": 4923
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4911,
              "end": 4923
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 4928,
                "end": 4940
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4928,
              "end": 4940
            }
          }
        ],
        "loc": {
          "start": 4434,
          "end": 4942
        }
      },
      "loc": {
        "start": 4425,
        "end": 4942
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 4943,
          "end": 4945
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4943,
        "end": 4945
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 4946,
          "end": 4956
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4946,
        "end": 4956
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 4957,
          "end": 4967
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4957,
        "end": 4967
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 4968,
          "end": 4978
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4968,
        "end": 4978
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 4979,
          "end": 4988
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4979,
        "end": 4988
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 4989,
          "end": 5000
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4989,
        "end": 5000
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 5001,
          "end": 5007
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
                "start": 5017,
                "end": 5027
              }
            },
            "directives": [],
            "loc": {
              "start": 5014,
              "end": 5027
            }
          }
        ],
        "loc": {
          "start": 5008,
          "end": 5029
        }
      },
      "loc": {
        "start": 5001,
        "end": 5029
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 5030,
          "end": 5035
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
                "value": "Organization",
                "loc": {
                  "start": 5049,
                  "end": 5061
                }
              },
              "loc": {
                "start": 5049,
                "end": 5061
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 5075,
                      "end": 5091
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5072,
                    "end": 5091
                  }
                }
              ],
              "loc": {
                "start": 5062,
                "end": 5097
              }
            },
            "loc": {
              "start": 5042,
              "end": 5097
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
                  "start": 5109,
                  "end": 5113
                }
              },
              "loc": {
                "start": 5109,
                "end": 5113
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
                      "start": 5127,
                      "end": 5135
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5124,
                    "end": 5135
                  }
                }
              ],
              "loc": {
                "start": 5114,
                "end": 5141
              }
            },
            "loc": {
              "start": 5102,
              "end": 5141
            }
          }
        ],
        "loc": {
          "start": 5036,
          "end": 5143
        }
      },
      "loc": {
        "start": 5030,
        "end": 5143
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 5144,
          "end": 5155
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5144,
        "end": 5155
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 5156,
          "end": 5170
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5156,
        "end": 5170
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 5171,
          "end": 5176
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5171,
        "end": 5176
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 5177,
          "end": 5186
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5177,
        "end": 5186
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 5187,
          "end": 5191
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
                "start": 5201,
                "end": 5209
              }
            },
            "directives": [],
            "loc": {
              "start": 5198,
              "end": 5209
            }
          }
        ],
        "loc": {
          "start": 5192,
          "end": 5211
        }
      },
      "loc": {
        "start": 5187,
        "end": 5211
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 5212,
          "end": 5226
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5212,
        "end": 5226
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 5227,
          "end": 5232
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5227,
        "end": 5232
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 5233,
          "end": 5236
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
                "start": 5243,
                "end": 5253
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5243,
              "end": 5253
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 5258,
                "end": 5267
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5258,
              "end": 5267
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 5272,
                "end": 5283
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5272,
              "end": 5283
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 5288,
                "end": 5297
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5288,
              "end": 5297
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 5302,
                "end": 5309
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5302,
              "end": 5309
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 5314,
                "end": 5322
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5314,
              "end": 5322
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 5327,
                "end": 5339
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5327,
              "end": 5339
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 5344,
                "end": 5352
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5344,
              "end": 5352
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 5357,
                "end": 5365
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5357,
              "end": 5365
            }
          }
        ],
        "loc": {
          "start": 5237,
          "end": 5367
        }
      },
      "loc": {
        "start": 5233,
        "end": 5367
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5404,
          "end": 5406
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5404,
        "end": 5406
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 5407,
          "end": 5417
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5407,
        "end": 5417
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5418,
          "end": 5427
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5418,
        "end": 5427
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 5477,
          "end": 5485
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
                "start": 5492,
                "end": 5504
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
                      "start": 5515,
                      "end": 5517
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5515,
                    "end": 5517
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 5526,
                      "end": 5534
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5526,
                    "end": 5534
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 5543,
                      "end": 5554
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5543,
                    "end": 5554
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 5563,
                      "end": 5575
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5563,
                    "end": 5575
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5584,
                      "end": 5588
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5584,
                    "end": 5588
                  }
                }
              ],
              "loc": {
                "start": 5505,
                "end": 5594
              }
            },
            "loc": {
              "start": 5492,
              "end": 5594
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5599,
                "end": 5601
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5599,
              "end": 5601
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5606,
                "end": 5616
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5606,
              "end": 5616
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5621,
                "end": 5631
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5621,
              "end": 5631
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 5636,
                "end": 5646
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5636,
              "end": 5646
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 5651,
                "end": 5660
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5651,
              "end": 5660
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 5665,
                "end": 5673
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5665,
              "end": 5673
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5678,
                "end": 5687
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5678,
              "end": 5687
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 5692,
                "end": 5699
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5692,
              "end": 5699
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contractType",
              "loc": {
                "start": 5704,
                "end": 5716
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5704,
              "end": 5716
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "content",
              "loc": {
                "start": 5721,
                "end": 5728
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5721,
              "end": 5728
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 5733,
                "end": 5745
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5733,
              "end": 5745
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 5750,
                "end": 5762
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5750,
              "end": 5762
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 5767,
                "end": 5780
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5767,
              "end": 5780
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 5785,
                "end": 5807
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5785,
              "end": 5807
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 5812,
                "end": 5822
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5812,
              "end": 5822
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 5827,
                "end": 5839
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5827,
              "end": 5839
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5844,
                "end": 5847
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
                      "start": 5858,
                      "end": 5868
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5858,
                    "end": 5868
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 5877,
                      "end": 5884
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5877,
                    "end": 5884
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5893,
                      "end": 5902
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5893,
                    "end": 5902
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 5911,
                      "end": 5920
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5911,
                    "end": 5920
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5929,
                      "end": 5938
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5929,
                    "end": 5938
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 5947,
                      "end": 5953
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5947,
                    "end": 5953
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5962,
                      "end": 5969
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5962,
                    "end": 5969
                  }
                }
              ],
              "loc": {
                "start": 5848,
                "end": 5975
              }
            },
            "loc": {
              "start": 5844,
              "end": 5975
            }
          }
        ],
        "loc": {
          "start": 5486,
          "end": 5977
        }
      },
      "loc": {
        "start": 5477,
        "end": 5977
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5978,
          "end": 5980
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5978,
        "end": 5980
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 5981,
          "end": 5991
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5981,
        "end": 5991
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 5992,
          "end": 6002
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5992,
        "end": 6002
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6003,
          "end": 6012
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6003,
        "end": 6012
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 6013,
          "end": 6024
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6013,
        "end": 6024
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 6025,
          "end": 6031
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
                "start": 6041,
                "end": 6051
              }
            },
            "directives": [],
            "loc": {
              "start": 6038,
              "end": 6051
            }
          }
        ],
        "loc": {
          "start": 6032,
          "end": 6053
        }
      },
      "loc": {
        "start": 6025,
        "end": 6053
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 6054,
          "end": 6059
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
                "value": "Organization",
                "loc": {
                  "start": 6073,
                  "end": 6085
                }
              },
              "loc": {
                "start": 6073,
                "end": 6085
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 6099,
                      "end": 6115
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6096,
                    "end": 6115
                  }
                }
              ],
              "loc": {
                "start": 6086,
                "end": 6121
              }
            },
            "loc": {
              "start": 6066,
              "end": 6121
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
                  "start": 6133,
                  "end": 6137
                }
              },
              "loc": {
                "start": 6133,
                "end": 6137
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
                      "start": 6151,
                      "end": 6159
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6148,
                    "end": 6159
                  }
                }
              ],
              "loc": {
                "start": 6138,
                "end": 6165
              }
            },
            "loc": {
              "start": 6126,
              "end": 6165
            }
          }
        ],
        "loc": {
          "start": 6060,
          "end": 6167
        }
      },
      "loc": {
        "start": 6054,
        "end": 6167
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 6168,
          "end": 6179
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6168,
        "end": 6179
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 6180,
          "end": 6194
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6180,
        "end": 6194
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 6195,
          "end": 6200
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6195,
        "end": 6200
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6201,
          "end": 6210
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6201,
        "end": 6210
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 6211,
          "end": 6215
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
                "start": 6225,
                "end": 6233
              }
            },
            "directives": [],
            "loc": {
              "start": 6222,
              "end": 6233
            }
          }
        ],
        "loc": {
          "start": 6216,
          "end": 6235
        }
      },
      "loc": {
        "start": 6211,
        "end": 6235
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 6236,
          "end": 6250
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6236,
        "end": 6250
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 6251,
          "end": 6256
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6251,
        "end": 6256
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6257,
          "end": 6260
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
                "start": 6267,
                "end": 6276
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6267,
              "end": 6276
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 6281,
                "end": 6292
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6281,
              "end": 6292
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 6297,
                "end": 6308
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6297,
              "end": 6308
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 6313,
                "end": 6322
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6313,
              "end": 6322
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 6327,
                "end": 6334
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6327,
              "end": 6334
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 6339,
                "end": 6347
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6339,
              "end": 6347
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6352,
                "end": 6364
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6352,
              "end": 6364
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 6369,
                "end": 6377
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6369,
              "end": 6377
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 6382,
                "end": 6390
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6382,
              "end": 6390
            }
          }
        ],
        "loc": {
          "start": 6261,
          "end": 6392
        }
      },
      "loc": {
        "start": 6257,
        "end": 6392
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6441,
          "end": 6443
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6441,
        "end": 6443
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6444,
          "end": 6453
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6444,
        "end": 6453
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 6493,
          "end": 6501
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
                "start": 6508,
                "end": 6520
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
                      "start": 6531,
                      "end": 6533
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6531,
                    "end": 6533
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6542,
                      "end": 6550
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6542,
                    "end": 6550
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 6559,
                      "end": 6570
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6559,
                    "end": 6570
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 6579,
                      "end": 6591
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6579,
                    "end": 6591
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6600,
                      "end": 6604
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6600,
                    "end": 6604
                  }
                }
              ],
              "loc": {
                "start": 6521,
                "end": 6610
              }
            },
            "loc": {
              "start": 6508,
              "end": 6610
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6615,
                "end": 6617
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6615,
              "end": 6617
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6622,
                "end": 6632
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6622,
              "end": 6632
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6637,
                "end": 6647
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6637,
              "end": 6647
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 6652,
                "end": 6662
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6652,
              "end": 6662
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isFile",
              "loc": {
                "start": 6667,
                "end": 6673
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6667,
              "end": 6673
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 6678,
                "end": 6686
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6678,
              "end": 6686
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6691,
                "end": 6700
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6691,
              "end": 6700
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 6705,
                "end": 6712
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6705,
              "end": 6712
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardType",
              "loc": {
                "start": 6717,
                "end": 6729
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6717,
              "end": 6729
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "props",
              "loc": {
                "start": 6734,
                "end": 6739
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6734,
              "end": 6739
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yup",
              "loc": {
                "start": 6744,
                "end": 6747
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6744,
              "end": 6747
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 6752,
                "end": 6764
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6752,
              "end": 6764
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 6769,
                "end": 6781
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6769,
              "end": 6781
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 6786,
                "end": 6799
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6786,
              "end": 6799
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 6804,
                "end": 6826
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6804,
              "end": 6826
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 6831,
                "end": 6841
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6831,
              "end": 6841
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 6846,
                "end": 6858
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6846,
              "end": 6858
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6863,
                "end": 6866
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
                      "start": 6877,
                      "end": 6887
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6877,
                    "end": 6887
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 6896,
                      "end": 6903
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6896,
                    "end": 6903
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6912,
                      "end": 6921
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6912,
                    "end": 6921
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 6930,
                      "end": 6939
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6930,
                    "end": 6939
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6948,
                      "end": 6957
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6948,
                    "end": 6957
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 6966,
                      "end": 6972
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6966,
                    "end": 6972
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6981,
                      "end": 6988
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6981,
                    "end": 6988
                  }
                }
              ],
              "loc": {
                "start": 6867,
                "end": 6994
              }
            },
            "loc": {
              "start": 6863,
              "end": 6994
            }
          }
        ],
        "loc": {
          "start": 6502,
          "end": 6996
        }
      },
      "loc": {
        "start": 6493,
        "end": 6996
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6997,
          "end": 6999
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6997,
        "end": 6999
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7000,
          "end": 7010
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7000,
        "end": 7010
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7011,
          "end": 7021
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7011,
        "end": 7021
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 7022,
          "end": 7031
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7022,
        "end": 7031
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 7032,
          "end": 7043
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7032,
        "end": 7043
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 7044,
          "end": 7050
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
                "start": 7060,
                "end": 7070
              }
            },
            "directives": [],
            "loc": {
              "start": 7057,
              "end": 7070
            }
          }
        ],
        "loc": {
          "start": 7051,
          "end": 7072
        }
      },
      "loc": {
        "start": 7044,
        "end": 7072
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 7073,
          "end": 7078
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
                "value": "Organization",
                "loc": {
                  "start": 7092,
                  "end": 7104
                }
              },
              "loc": {
                "start": 7092,
                "end": 7104
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 7118,
                      "end": 7134
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7115,
                    "end": 7134
                  }
                }
              ],
              "loc": {
                "start": 7105,
                "end": 7140
              }
            },
            "loc": {
              "start": 7085,
              "end": 7140
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
                  "start": 7152,
                  "end": 7156
                }
              },
              "loc": {
                "start": 7152,
                "end": 7156
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
                      "start": 7170,
                      "end": 7178
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7167,
                    "end": 7178
                  }
                }
              ],
              "loc": {
                "start": 7157,
                "end": 7184
              }
            },
            "loc": {
              "start": 7145,
              "end": 7184
            }
          }
        ],
        "loc": {
          "start": 7079,
          "end": 7186
        }
      },
      "loc": {
        "start": 7073,
        "end": 7186
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 7187,
          "end": 7198
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7187,
        "end": 7198
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 7199,
          "end": 7213
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7199,
        "end": 7213
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 7214,
          "end": 7219
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7214,
        "end": 7219
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7220,
          "end": 7229
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7220,
        "end": 7229
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 7230,
          "end": 7234
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
                "start": 7244,
                "end": 7252
              }
            },
            "directives": [],
            "loc": {
              "start": 7241,
              "end": 7252
            }
          }
        ],
        "loc": {
          "start": 7235,
          "end": 7254
        }
      },
      "loc": {
        "start": 7230,
        "end": 7254
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 7255,
          "end": 7269
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7255,
        "end": 7269
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 7270,
          "end": 7275
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7270,
        "end": 7275
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7276,
          "end": 7279
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
                "start": 7286,
                "end": 7295
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7286,
              "end": 7295
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 7300,
                "end": 7311
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7300,
              "end": 7311
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 7316,
                "end": 7327
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7316,
              "end": 7327
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7332,
                "end": 7341
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7332,
              "end": 7341
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 7346,
                "end": 7353
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7346,
              "end": 7353
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 7358,
                "end": 7366
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7358,
              "end": 7366
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7371,
                "end": 7383
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7371,
              "end": 7383
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7388,
                "end": 7396
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7388,
              "end": 7396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 7401,
                "end": 7409
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7401,
              "end": 7409
            }
          }
        ],
        "loc": {
          "start": 7280,
          "end": 7411
        }
      },
      "loc": {
        "start": 7276,
        "end": 7411
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7450,
          "end": 7452
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7450,
        "end": 7452
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 7453,
          "end": 7462
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7453,
        "end": 7462
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7492,
          "end": 7494
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7492,
        "end": 7494
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7495,
          "end": 7505
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7495,
        "end": 7505
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 7506,
          "end": 7509
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7506,
        "end": 7509
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7510,
          "end": 7519
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7510,
        "end": 7519
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 7520,
          "end": 7532
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
                "start": 7539,
                "end": 7541
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7539,
              "end": 7541
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 7546,
                "end": 7554
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7546,
              "end": 7554
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 7559,
                "end": 7570
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7559,
              "end": 7570
            }
          }
        ],
        "loc": {
          "start": 7533,
          "end": 7572
        }
      },
      "loc": {
        "start": 7520,
        "end": 7572
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7573,
          "end": 7576
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
                "start": 7583,
                "end": 7588
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7583,
              "end": 7588
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7593,
                "end": 7605
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7593,
              "end": 7605
            }
          }
        ],
        "loc": {
          "start": 7577,
          "end": 7607
        }
      },
      "loc": {
        "start": 7573,
        "end": 7607
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 7639,
          "end": 7651
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
                "start": 7658,
                "end": 7660
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7658,
              "end": 7660
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 7665,
                "end": 7673
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7665,
              "end": 7673
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 7678,
                "end": 7681
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7678,
              "end": 7681
            }
          }
        ],
        "loc": {
          "start": 7652,
          "end": 7683
        }
      },
      "loc": {
        "start": 7639,
        "end": 7683
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7684,
          "end": 7686
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7684,
        "end": 7686
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7687,
          "end": 7697
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7687,
        "end": 7697
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7698,
          "end": 7708
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7698,
        "end": 7708
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7709,
          "end": 7720
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7709,
        "end": 7720
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7721,
          "end": 7727
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7721,
        "end": 7727
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7728,
          "end": 7733
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7728,
        "end": 7733
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7734,
          "end": 7738
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7734,
        "end": 7738
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7739,
          "end": 7751
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7739,
        "end": 7751
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7752,
          "end": 7761
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7752,
        "end": 7761
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsReceivedCount",
        "loc": {
          "start": 7762,
          "end": 7782
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7762,
        "end": 7782
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7783,
          "end": 7786
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
                "start": 7793,
                "end": 7802
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7793,
              "end": 7802
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 7807,
                "end": 7816
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7807,
              "end": 7816
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7821,
                "end": 7830
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7821,
              "end": 7830
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7835,
                "end": 7847
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7835,
              "end": 7847
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7852,
                "end": 7860
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7852,
              "end": 7860
            }
          }
        ],
        "loc": {
          "start": 7787,
          "end": 7862
        }
      },
      "loc": {
        "start": 7783,
        "end": 7862
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7893,
          "end": 7895
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7893,
        "end": 7895
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7896,
          "end": 7906
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7896,
        "end": 7906
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7907,
          "end": 7917
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7907,
        "end": 7917
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7918,
          "end": 7929
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7918,
        "end": 7929
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7930,
          "end": 7936
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7930,
        "end": 7936
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7937,
          "end": 7942
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7937,
        "end": 7942
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7943,
          "end": 7947
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7943,
        "end": 7947
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7948,
          "end": 7960
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7948,
        "end": 7960
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
                      "value": "Organization",
                      "loc": {
                        "start": 553,
                        "end": 565
                      }
                    },
                    "loc": {
                      "start": 553,
                      "end": 565
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
                          "value": "Organization_nav",
                          "loc": {
                            "start": 579,
                            "end": 595
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 576,
                          "end": 595
                        }
                      }
                    ],
                    "loc": {
                      "start": 566,
                      "end": 601
                    }
                  },
                  "loc": {
                    "start": 546,
                    "end": 601
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
                        "start": 613,
                        "end": 617
                      }
                    },
                    "loc": {
                      "start": 613,
                      "end": 617
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
                            "start": 631,
                            "end": 639
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 628,
                          "end": 639
                        }
                      }
                    ],
                    "loc": {
                      "start": 618,
                      "end": 645
                    }
                  },
                  "loc": {
                    "start": 606,
                    "end": 645
                  }
                }
              ],
              "loc": {
                "start": 540,
                "end": 647
              }
            },
            "loc": {
              "start": 534,
              "end": 647
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 648,
                "end": 659
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 648,
              "end": 659
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 660,
                "end": 674
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 660,
              "end": 674
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 675,
                "end": 680
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 675,
              "end": 680
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 681,
                "end": 690
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 681,
              "end": 690
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 691,
                "end": 695
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
                      "start": 705,
                      "end": 713
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 702,
                    "end": 713
                  }
                }
              ],
              "loc": {
                "start": 696,
                "end": 715
              }
            },
            "loc": {
              "start": 691,
              "end": 715
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 716,
                "end": 730
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 716,
              "end": 730
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 731,
                "end": 736
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 731,
              "end": 736
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 737,
                "end": 740
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
                      "start": 747,
                      "end": 756
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 747,
                    "end": 756
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
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
                    "value": "canTransfer",
                    "loc": {
                      "start": 777,
                      "end": 788
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 777,
                    "end": 788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 793,
                      "end": 802
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 793,
                    "end": 802
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 807,
                      "end": 814
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 807,
                    "end": 814
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 819,
                      "end": 827
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 819,
                    "end": 827
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 832,
                      "end": 844
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 832,
                    "end": 844
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 849,
                      "end": 857
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 849,
                    "end": 857
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 862,
                      "end": 870
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 862,
                    "end": 870
                  }
                }
              ],
              "loc": {
                "start": 741,
                "end": 872
              }
            },
            "loc": {
              "start": 737,
              "end": 872
            }
          }
        ],
        "loc": {
          "start": 26,
          "end": 874
        }
      },
      "loc": {
        "start": 1,
        "end": 874
      }
    },
    "Api_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Api_nav",
        "loc": {
          "start": 884,
          "end": 891
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 895,
            "end": 898
          }
        },
        "loc": {
          "start": 895,
          "end": 898
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
                "start": 901,
                "end": 903
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 901,
              "end": 903
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 904,
                "end": 913
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 904,
              "end": 913
            }
          }
        ],
        "loc": {
          "start": 899,
          "end": 915
        }
      },
      "loc": {
        "start": 875,
        "end": 915
      }
    },
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 925,
          "end": 935
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 939,
            "end": 944
          }
        },
        "loc": {
          "start": 939,
          "end": 944
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
                "start": 947,
                "end": 949
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 947,
              "end": 949
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 950,
                "end": 960
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 950,
              "end": 960
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 961,
                "end": 971
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 961,
              "end": 971
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 972,
                "end": 977
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 972,
              "end": 977
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 978,
                "end": 983
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 978,
              "end": 983
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 984,
                "end": 989
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
                      "value": "Organization",
                      "loc": {
                        "start": 1003,
                        "end": 1015
                      }
                    },
                    "loc": {
                      "start": 1003,
                      "end": 1015
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
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1029,
                            "end": 1045
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1026,
                          "end": 1045
                        }
                      }
                    ],
                    "loc": {
                      "start": 1016,
                      "end": 1051
                    }
                  },
                  "loc": {
                    "start": 996,
                    "end": 1051
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
                        "start": 1063,
                        "end": 1067
                      }
                    },
                    "loc": {
                      "start": 1063,
                      "end": 1067
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
                            "start": 1081,
                            "end": 1089
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1078,
                          "end": 1089
                        }
                      }
                    ],
                    "loc": {
                      "start": 1068,
                      "end": 1095
                    }
                  },
                  "loc": {
                    "start": 1056,
                    "end": 1095
                  }
                }
              ],
              "loc": {
                "start": 990,
                "end": 1097
              }
            },
            "loc": {
              "start": 984,
              "end": 1097
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1098,
                "end": 1101
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
                      "start": 1108,
                      "end": 1117
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1108,
                    "end": 1117
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1122,
                      "end": 1131
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1122,
                    "end": 1131
                  }
                }
              ],
              "loc": {
                "start": 1102,
                "end": 1133
              }
            },
            "loc": {
              "start": 1098,
              "end": 1133
            }
          }
        ],
        "loc": {
          "start": 945,
          "end": 1135
        }
      },
      "loc": {
        "start": 916,
        "end": 1135
      }
    },
    "Note_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_list",
        "loc": {
          "start": 1145,
          "end": 1154
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 1158,
            "end": 1162
          }
        },
        "loc": {
          "start": 1158,
          "end": 1162
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
                "start": 1165,
                "end": 1173
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
                      "start": 1180,
                      "end": 1192
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
                            "start": 1203,
                            "end": 1205
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1203,
                          "end": 1205
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1214,
                            "end": 1222
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1214,
                          "end": 1222
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1231,
                            "end": 1242
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1231,
                          "end": 1242
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1251,
                            "end": 1255
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1251,
                          "end": 1255
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pages",
                          "loc": {
                            "start": 1264,
                            "end": 1269
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
                                  "start": 1284,
                                  "end": 1286
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1284,
                                "end": 1286
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "pageIndex",
                                "loc": {
                                  "start": 1299,
                                  "end": 1308
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1299,
                                "end": 1308
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "text",
                                "loc": {
                                  "start": 1321,
                                  "end": 1325
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1321,
                                "end": 1325
                              }
                            }
                          ],
                          "loc": {
                            "start": 1270,
                            "end": 1335
                          }
                        },
                        "loc": {
                          "start": 1264,
                          "end": 1335
                        }
                      }
                    ],
                    "loc": {
                      "start": 1193,
                      "end": 1341
                    }
                  },
                  "loc": {
                    "start": 1180,
                    "end": 1341
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1346,
                      "end": 1348
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1346,
                    "end": 1348
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1353,
                      "end": 1363
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1353,
                    "end": 1363
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1368,
                      "end": 1378
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1368,
                    "end": 1378
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1383,
                      "end": 1391
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1383,
                    "end": 1391
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1396,
                      "end": 1405
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1396,
                    "end": 1405
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1410,
                      "end": 1422
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1410,
                    "end": 1422
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1427,
                      "end": 1439
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1427,
                    "end": 1439
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1444,
                      "end": 1456
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1444,
                    "end": 1456
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 1461,
                      "end": 1464
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
                            "start": 1475,
                            "end": 1485
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1475,
                          "end": 1485
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 1494,
                            "end": 1501
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1494,
                          "end": 1501
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 1510,
                            "end": 1519
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1510,
                          "end": 1519
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 1528,
                            "end": 1537
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1528,
                          "end": 1537
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 1546,
                            "end": 1555
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1546,
                          "end": 1555
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 1564,
                            "end": 1570
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1564,
                          "end": 1570
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 1579,
                            "end": 1586
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1579,
                          "end": 1586
                        }
                      }
                    ],
                    "loc": {
                      "start": 1465,
                      "end": 1592
                    }
                  },
                  "loc": {
                    "start": 1461,
                    "end": 1592
                  }
                }
              ],
              "loc": {
                "start": 1174,
                "end": 1594
              }
            },
            "loc": {
              "start": 1165,
              "end": 1594
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1595,
                "end": 1597
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1595,
              "end": 1597
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1598,
                "end": 1608
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1598,
              "end": 1608
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1609,
                "end": 1619
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1609,
              "end": 1619
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1620,
                "end": 1629
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1620,
              "end": 1629
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1630,
                "end": 1641
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1630,
              "end": 1641
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1642,
                "end": 1648
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
                      "start": 1658,
                      "end": 1668
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1655,
                    "end": 1668
                  }
                }
              ],
              "loc": {
                "start": 1649,
                "end": 1670
              }
            },
            "loc": {
              "start": 1642,
              "end": 1670
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1671,
                "end": 1676
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
                      "value": "Organization",
                      "loc": {
                        "start": 1690,
                        "end": 1702
                      }
                    },
                    "loc": {
                      "start": 1690,
                      "end": 1702
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
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1716,
                            "end": 1732
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1713,
                          "end": 1732
                        }
                      }
                    ],
                    "loc": {
                      "start": 1703,
                      "end": 1738
                    }
                  },
                  "loc": {
                    "start": 1683,
                    "end": 1738
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
                        "start": 1750,
                        "end": 1754
                      }
                    },
                    "loc": {
                      "start": 1750,
                      "end": 1754
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
                            "start": 1768,
                            "end": 1776
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1765,
                          "end": 1776
                        }
                      }
                    ],
                    "loc": {
                      "start": 1755,
                      "end": 1782
                    }
                  },
                  "loc": {
                    "start": 1743,
                    "end": 1782
                  }
                }
              ],
              "loc": {
                "start": 1677,
                "end": 1784
              }
            },
            "loc": {
              "start": 1671,
              "end": 1784
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1785,
                "end": 1796
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1785,
              "end": 1796
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1797,
                "end": 1811
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1797,
              "end": 1811
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1812,
                "end": 1817
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1812,
              "end": 1817
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1818,
                "end": 1827
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1818,
              "end": 1827
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1828,
                "end": 1832
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
                      "start": 1842,
                      "end": 1850
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1839,
                    "end": 1850
                  }
                }
              ],
              "loc": {
                "start": 1833,
                "end": 1852
              }
            },
            "loc": {
              "start": 1828,
              "end": 1852
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1853,
                "end": 1867
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1853,
              "end": 1867
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1868,
                "end": 1873
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1868,
              "end": 1873
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1874,
                "end": 1877
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
                      "start": 1884,
                      "end": 1893
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1884,
                    "end": 1893
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1898,
                      "end": 1909
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1898,
                    "end": 1909
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 1914,
                      "end": 1925
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1914,
                    "end": 1925
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1930,
                      "end": 1939
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1930,
                    "end": 1939
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1944,
                      "end": 1951
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1944,
                    "end": 1951
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1956,
                      "end": 1964
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1956,
                    "end": 1964
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1969,
                      "end": 1981
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1969,
                    "end": 1981
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1986,
                      "end": 1994
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1986,
                    "end": 1994
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1999,
                      "end": 2007
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1999,
                    "end": 2007
                  }
                }
              ],
              "loc": {
                "start": 1878,
                "end": 2009
              }
            },
            "loc": {
              "start": 1874,
              "end": 2009
            }
          }
        ],
        "loc": {
          "start": 1163,
          "end": 2011
        }
      },
      "loc": {
        "start": 1136,
        "end": 2011
      }
    },
    "Note_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_nav",
        "loc": {
          "start": 2021,
          "end": 2029
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 2033,
            "end": 2037
          }
        },
        "loc": {
          "start": 2033,
          "end": 2037
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
                "start": 2040,
                "end": 2042
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2040,
              "end": 2042
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2043,
                "end": 2052
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2043,
              "end": 2052
            }
          }
        ],
        "loc": {
          "start": 2038,
          "end": 2054
        }
      },
      "loc": {
        "start": 2012,
        "end": 2054
      }
    },
    "Organization_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_list",
        "loc": {
          "start": 2064,
          "end": 2081
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 2085,
            "end": 2097
          }
        },
        "loc": {
          "start": 2085,
          "end": 2097
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
                "start": 2100,
                "end": 2102
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2100,
              "end": 2102
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 2103,
                "end": 2114
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2103,
              "end": 2114
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2115,
                "end": 2121
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2115,
              "end": 2121
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2122,
                "end": 2132
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2122,
              "end": 2132
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2133,
                "end": 2143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2133,
              "end": 2143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOpenToNewMembers",
              "loc": {
                "start": 2144,
                "end": 2162
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2144,
              "end": 2162
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2163,
                "end": 2172
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2163,
              "end": 2172
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 2173,
                "end": 2186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2173,
              "end": 2186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membersCount",
              "loc": {
                "start": 2187,
                "end": 2199
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2187,
              "end": 2199
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 2200,
                "end": 2212
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2200,
              "end": 2212
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 2213,
                "end": 2225
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2213,
              "end": 2225
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2226,
                "end": 2235
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2226,
              "end": 2235
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2236,
                "end": 2240
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
                      "start": 2250,
                      "end": 2258
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2247,
                    "end": 2258
                  }
                }
              ],
              "loc": {
                "start": 2241,
                "end": 2260
              }
            },
            "loc": {
              "start": 2236,
              "end": 2260
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2261,
                "end": 2273
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
                      "start": 2280,
                      "end": 2282
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2280,
                    "end": 2282
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2287,
                      "end": 2295
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2287,
                    "end": 2295
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 2300,
                      "end": 2303
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2300,
                    "end": 2303
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2308,
                      "end": 2312
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2308,
                    "end": 2312
                  }
                }
              ],
              "loc": {
                "start": 2274,
                "end": 2314
              }
            },
            "loc": {
              "start": 2261,
              "end": 2314
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2315,
                "end": 2318
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
                      "start": 2325,
                      "end": 2338
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2325,
                    "end": 2338
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2343,
                      "end": 2352
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2343,
                    "end": 2352
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2357,
                      "end": 2368
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2357,
                    "end": 2368
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2373,
                      "end": 2382
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2373,
                    "end": 2382
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2387,
                      "end": 2396
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2387,
                    "end": 2396
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2401,
                      "end": 2408
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2401,
                    "end": 2408
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2413,
                      "end": 2425
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2413,
                    "end": 2425
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2430,
                      "end": 2438
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2430,
                    "end": 2438
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 2443,
                      "end": 2457
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
                            "start": 2468,
                            "end": 2470
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2468,
                          "end": 2470
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2479,
                            "end": 2489
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2479,
                          "end": 2489
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2498,
                            "end": 2508
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2498,
                          "end": 2508
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 2517,
                            "end": 2524
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2517,
                          "end": 2524
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 2533,
                            "end": 2544
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2533,
                          "end": 2544
                        }
                      }
                    ],
                    "loc": {
                      "start": 2458,
                      "end": 2550
                    }
                  },
                  "loc": {
                    "start": 2443,
                    "end": 2550
                  }
                }
              ],
              "loc": {
                "start": 2319,
                "end": 2552
              }
            },
            "loc": {
              "start": 2315,
              "end": 2552
            }
          }
        ],
        "loc": {
          "start": 2098,
          "end": 2554
        }
      },
      "loc": {
        "start": 2055,
        "end": 2554
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 2564,
          "end": 2580
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 2584,
            "end": 2596
          }
        },
        "loc": {
          "start": 2584,
          "end": 2596
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
                "start": 2599,
                "end": 2601
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2599,
              "end": 2601
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 2602,
                "end": 2613
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2602,
              "end": 2613
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2614,
                "end": 2620
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2614,
              "end": 2620
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 2621,
                "end": 2633
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2621,
              "end": 2633
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2634,
                "end": 2637
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
                      "start": 2644,
                      "end": 2657
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2644,
                    "end": 2657
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2662,
                      "end": 2671
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2662,
                    "end": 2671
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2676,
                      "end": 2687
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2676,
                    "end": 2687
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2692,
                      "end": 2701
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2692,
                    "end": 2701
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2706,
                      "end": 2715
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2706,
                    "end": 2715
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2720,
                      "end": 2727
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2720,
                    "end": 2727
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2732,
                      "end": 2744
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2732,
                    "end": 2744
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2749,
                      "end": 2757
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2749,
                    "end": 2757
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 2762,
                      "end": 2776
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
                            "start": 2787,
                            "end": 2789
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2787,
                          "end": 2789
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2798,
                            "end": 2808
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2798,
                          "end": 2808
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2817,
                            "end": 2827
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2817,
                          "end": 2827
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 2836,
                            "end": 2843
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2836,
                          "end": 2843
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 2852,
                            "end": 2863
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2852,
                          "end": 2863
                        }
                      }
                    ],
                    "loc": {
                      "start": 2777,
                      "end": 2869
                    }
                  },
                  "loc": {
                    "start": 2762,
                    "end": 2869
                  }
                }
              ],
              "loc": {
                "start": 2638,
                "end": 2871
              }
            },
            "loc": {
              "start": 2634,
              "end": 2871
            }
          }
        ],
        "loc": {
          "start": 2597,
          "end": 2873
        }
      },
      "loc": {
        "start": 2555,
        "end": 2873
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 2883,
          "end": 2895
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 2899,
            "end": 2906
          }
        },
        "loc": {
          "start": 2899,
          "end": 2906
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
                "start": 2909,
                "end": 2917
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
                      "start": 2924,
                      "end": 2936
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
                            "start": 2947,
                            "end": 2949
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2947,
                          "end": 2949
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 2958,
                            "end": 2966
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2958,
                          "end": 2966
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2975,
                            "end": 2986
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2975,
                          "end": 2986
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2995,
                            "end": 2999
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2995,
                          "end": 2999
                        }
                      }
                    ],
                    "loc": {
                      "start": 2937,
                      "end": 3005
                    }
                  },
                  "loc": {
                    "start": 2924,
                    "end": 3005
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3010,
                      "end": 3012
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3010,
                    "end": 3012
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3017,
                      "end": 3027
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3017,
                    "end": 3027
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3032,
                      "end": 3042
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3032,
                    "end": 3042
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 3047,
                      "end": 3063
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3047,
                    "end": 3063
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 3068,
                      "end": 3076
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3068,
                    "end": 3076
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 3081,
                      "end": 3090
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3081,
                    "end": 3090
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 3095,
                      "end": 3107
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3095,
                    "end": 3107
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 3112,
                      "end": 3128
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3112,
                    "end": 3128
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 3133,
                      "end": 3143
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3133,
                    "end": 3143
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 3148,
                      "end": 3160
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3148,
                    "end": 3160
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 3165,
                      "end": 3177
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3165,
                    "end": 3177
                  }
                }
              ],
              "loc": {
                "start": 2918,
                "end": 3179
              }
            },
            "loc": {
              "start": 2909,
              "end": 3179
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3180,
                "end": 3182
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3180,
              "end": 3182
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3183,
                "end": 3193
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3183,
              "end": 3193
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3194,
                "end": 3204
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3194,
              "end": 3204
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3205,
                "end": 3214
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3205,
              "end": 3214
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 3215,
                "end": 3226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3215,
              "end": 3226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 3227,
                "end": 3233
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
                      "start": 3243,
                      "end": 3253
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3240,
                    "end": 3253
                  }
                }
              ],
              "loc": {
                "start": 3234,
                "end": 3255
              }
            },
            "loc": {
              "start": 3227,
              "end": 3255
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 3256,
                "end": 3261
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
                      "value": "Organization",
                      "loc": {
                        "start": 3275,
                        "end": 3287
                      }
                    },
                    "loc": {
                      "start": 3275,
                      "end": 3287
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
                          "value": "Organization_nav",
                          "loc": {
                            "start": 3301,
                            "end": 3317
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3298,
                          "end": 3317
                        }
                      }
                    ],
                    "loc": {
                      "start": 3288,
                      "end": 3323
                    }
                  },
                  "loc": {
                    "start": 3268,
                    "end": 3323
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
                        "start": 3335,
                        "end": 3339
                      }
                    },
                    "loc": {
                      "start": 3335,
                      "end": 3339
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
                            "start": 3353,
                            "end": 3361
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3350,
                          "end": 3361
                        }
                      }
                    ],
                    "loc": {
                      "start": 3340,
                      "end": 3367
                    }
                  },
                  "loc": {
                    "start": 3328,
                    "end": 3367
                  }
                }
              ],
              "loc": {
                "start": 3262,
                "end": 3369
              }
            },
            "loc": {
              "start": 3256,
              "end": 3369
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 3370,
                "end": 3381
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3370,
              "end": 3381
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 3382,
                "end": 3396
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3382,
              "end": 3396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 3397,
                "end": 3402
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3397,
              "end": 3402
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 3403,
                "end": 3412
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3403,
              "end": 3412
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 3413,
                "end": 3417
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
                      "start": 3427,
                      "end": 3435
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3424,
                    "end": 3435
                  }
                }
              ],
              "loc": {
                "start": 3418,
                "end": 3437
              }
            },
            "loc": {
              "start": 3413,
              "end": 3437
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 3438,
                "end": 3452
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3438,
              "end": 3452
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 3453,
                "end": 3458
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3453,
              "end": 3458
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 3459,
                "end": 3462
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
                      "start": 3469,
                      "end": 3478
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3469,
                    "end": 3478
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 3483,
                      "end": 3494
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3483,
                    "end": 3494
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 3499,
                      "end": 3510
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3499,
                    "end": 3510
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 3515,
                      "end": 3524
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3515,
                    "end": 3524
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 3529,
                      "end": 3536
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3529,
                    "end": 3536
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 3541,
                      "end": 3549
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3541,
                    "end": 3549
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 3554,
                      "end": 3566
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3554,
                    "end": 3566
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 3571,
                      "end": 3579
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3571,
                    "end": 3579
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 3584,
                      "end": 3592
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3584,
                    "end": 3592
                  }
                }
              ],
              "loc": {
                "start": 3463,
                "end": 3594
              }
            },
            "loc": {
              "start": 3459,
              "end": 3594
            }
          }
        ],
        "loc": {
          "start": 2907,
          "end": 3596
        }
      },
      "loc": {
        "start": 2874,
        "end": 3596
      }
    },
    "Project_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_nav",
        "loc": {
          "start": 3606,
          "end": 3617
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3621,
            "end": 3628
          }
        },
        "loc": {
          "start": 3621,
          "end": 3628
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
                "start": 3631,
                "end": 3633
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3631,
              "end": 3633
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3634,
                "end": 3643
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3634,
              "end": 3643
            }
          }
        ],
        "loc": {
          "start": 3629,
          "end": 3645
        }
      },
      "loc": {
        "start": 3597,
        "end": 3645
      }
    },
    "Question_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Question_list",
        "loc": {
          "start": 3655,
          "end": 3668
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Question",
          "loc": {
            "start": 3672,
            "end": 3680
          }
        },
        "loc": {
          "start": 3672,
          "end": 3680
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
                "start": 3683,
                "end": 3695
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
                      "start": 3702,
                      "end": 3704
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3702,
                    "end": 3704
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3709,
                      "end": 3717
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3709,
                    "end": 3717
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3722,
                      "end": 3733
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3722,
                    "end": 3733
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3738,
                      "end": 3742
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3738,
                    "end": 3742
                  }
                }
              ],
              "loc": {
                "start": 3696,
                "end": 3744
              }
            },
            "loc": {
              "start": 3683,
              "end": 3744
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3745,
                "end": 3747
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3745,
              "end": 3747
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3748,
                "end": 3758
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3748,
              "end": 3758
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3759,
                "end": 3769
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3759,
              "end": 3769
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "createdBy",
              "loc": {
                "start": 3770,
                "end": 3779
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
                      "start": 3786,
                      "end": 3788
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3786,
                    "end": 3788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3793,
                      "end": 3803
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3793,
                    "end": 3803
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3808,
                      "end": 3818
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3808,
                    "end": 3818
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
                    "loc": {
                      "start": 3823,
                      "end": 3834
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3823,
                    "end": 3834
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 3839,
                      "end": 3845
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3839,
                    "end": 3845
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBot",
                    "loc": {
                      "start": 3850,
                      "end": 3855
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3850,
                    "end": 3855
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3860,
                      "end": 3864
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3860,
                    "end": 3864
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 3869,
                      "end": 3881
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3869,
                    "end": 3881
                  }
                }
              ],
              "loc": {
                "start": 3780,
                "end": 3883
              }
            },
            "loc": {
              "start": 3770,
              "end": 3883
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "hasAcceptedAnswer",
              "loc": {
                "start": 3884,
                "end": 3901
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3884,
              "end": 3901
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3902,
                "end": 3911
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3902,
              "end": 3911
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 3912,
                "end": 3917
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3912,
              "end": 3917
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 3918,
                "end": 3927
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3918,
              "end": 3927
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "answersCount",
              "loc": {
                "start": 3928,
                "end": 3940
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3928,
              "end": 3940
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 3941,
                "end": 3954
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3941,
              "end": 3954
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 3955,
                "end": 3967
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3955,
              "end": 3967
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forObject",
              "loc": {
                "start": 3968,
                "end": 3977
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
                        "start": 3991,
                        "end": 3994
                      }
                    },
                    "loc": {
                      "start": 3991,
                      "end": 3994
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
                            "start": 4008,
                            "end": 4015
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4005,
                          "end": 4015
                        }
                      }
                    ],
                    "loc": {
                      "start": 3995,
                      "end": 4021
                    }
                  },
                  "loc": {
                    "start": 3984,
                    "end": 4021
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
                        "start": 4033,
                        "end": 4037
                      }
                    },
                    "loc": {
                      "start": 4033,
                      "end": 4037
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
                            "start": 4051,
                            "end": 4059
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4048,
                          "end": 4059
                        }
                      }
                    ],
                    "loc": {
                      "start": 4038,
                      "end": 4065
                    }
                  },
                  "loc": {
                    "start": 4026,
                    "end": 4065
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 4077,
                        "end": 4089
                      }
                    },
                    "loc": {
                      "start": 4077,
                      "end": 4089
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
                          "value": "Organization_nav",
                          "loc": {
                            "start": 4103,
                            "end": 4119
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4100,
                          "end": 4119
                        }
                      }
                    ],
                    "loc": {
                      "start": 4090,
                      "end": 4125
                    }
                  },
                  "loc": {
                    "start": 4070,
                    "end": 4125
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
                        "start": 4137,
                        "end": 4144
                      }
                    },
                    "loc": {
                      "start": 4137,
                      "end": 4144
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
                            "start": 4158,
                            "end": 4169
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4155,
                          "end": 4169
                        }
                      }
                    ],
                    "loc": {
                      "start": 4145,
                      "end": 4175
                    }
                  },
                  "loc": {
                    "start": 4130,
                    "end": 4175
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
                        "start": 4187,
                        "end": 4194
                      }
                    },
                    "loc": {
                      "start": 4187,
                      "end": 4194
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
                            "start": 4208,
                            "end": 4219
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4205,
                          "end": 4219
                        }
                      }
                    ],
                    "loc": {
                      "start": 4195,
                      "end": 4225
                    }
                  },
                  "loc": {
                    "start": 4180,
                    "end": 4225
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "SmartContract",
                      "loc": {
                        "start": 4237,
                        "end": 4250
                      }
                    },
                    "loc": {
                      "start": 4237,
                      "end": 4250
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
                          "value": "SmartContract_nav",
                          "loc": {
                            "start": 4264,
                            "end": 4281
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4261,
                          "end": 4281
                        }
                      }
                    ],
                    "loc": {
                      "start": 4251,
                      "end": 4287
                    }
                  },
                  "loc": {
                    "start": 4230,
                    "end": 4287
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
                        "start": 4299,
                        "end": 4307
                      }
                    },
                    "loc": {
                      "start": 4299,
                      "end": 4307
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
                            "start": 4321,
                            "end": 4333
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4318,
                          "end": 4333
                        }
                      }
                    ],
                    "loc": {
                      "start": 4308,
                      "end": 4339
                    }
                  },
                  "loc": {
                    "start": 4292,
                    "end": 4339
                  }
                }
              ],
              "loc": {
                "start": 3978,
                "end": 4341
              }
            },
            "loc": {
              "start": 3968,
              "end": 4341
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 4342,
                "end": 4346
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
                      "start": 4356,
                      "end": 4364
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4353,
                    "end": 4364
                  }
                }
              ],
              "loc": {
                "start": 4347,
                "end": 4366
              }
            },
            "loc": {
              "start": 4342,
              "end": 4366
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 4367,
                "end": 4370
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
                      "start": 4377,
                      "end": 4385
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4377,
                    "end": 4385
                  }
                }
              ],
              "loc": {
                "start": 4371,
                "end": 4387
              }
            },
            "loc": {
              "start": 4367,
              "end": 4387
            }
          }
        ],
        "loc": {
          "start": 3681,
          "end": 4389
        }
      },
      "loc": {
        "start": 3646,
        "end": 4389
      }
    },
    "Routine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_list",
        "loc": {
          "start": 4399,
          "end": 4411
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 4415,
            "end": 4422
          }
        },
        "loc": {
          "start": 4415,
          "end": 4422
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
                "start": 4425,
                "end": 4433
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
                      "start": 4440,
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
                          "value": "id",
                          "loc": {
                            "start": 4463,
                            "end": 4465
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4463,
                          "end": 4465
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 4474,
                            "end": 4482
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4474,
                          "end": 4482
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 4491,
                            "end": 4502
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4491,
                          "end": 4502
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 4511,
                            "end": 4523
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4511,
                          "end": 4523
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4532,
                            "end": 4536
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4532,
                          "end": 4536
                        }
                      }
                    ],
                    "loc": {
                      "start": 4453,
                      "end": 4542
                    }
                  },
                  "loc": {
                    "start": 4440,
                    "end": 4542
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4547,
                      "end": 4549
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4547,
                    "end": 4549
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4554,
                      "end": 4564
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4554,
                    "end": 4564
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4569,
                      "end": 4579
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4569,
                    "end": 4579
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 4584,
                      "end": 4595
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4584,
                    "end": 4595
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 4600,
                      "end": 4613
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4600,
                    "end": 4613
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 4618,
                      "end": 4628
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4618,
                    "end": 4628
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 4633,
                      "end": 4642
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4633,
                    "end": 4642
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 4647,
                      "end": 4655
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4647,
                    "end": 4655
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 4660,
                      "end": 4669
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4660,
                    "end": 4669
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 4674,
                      "end": 4684
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4674,
                    "end": 4684
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 4689,
                      "end": 4701
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4689,
                    "end": 4701
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 4706,
                      "end": 4720
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4706,
                    "end": 4720
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractCallData",
                    "loc": {
                      "start": 4725,
                      "end": 4746
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4725,
                    "end": 4746
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apiCallData",
                    "loc": {
                      "start": 4751,
                      "end": 4762
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4751,
                    "end": 4762
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 4767,
                      "end": 4779
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4767,
                    "end": 4779
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 4784,
                      "end": 4796
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4784,
                    "end": 4796
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 4801,
                      "end": 4814
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4801,
                    "end": 4814
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 4819,
                      "end": 4841
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4819,
                    "end": 4841
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 4846,
                      "end": 4856
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4846,
                    "end": 4856
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 4861,
                      "end": 4872
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4861,
                    "end": 4872
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 4877,
                      "end": 4887
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4877,
                    "end": 4887
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 4892,
                      "end": 4906
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4892,
                    "end": 4906
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 4911,
                      "end": 4923
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4911,
                    "end": 4923
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 4928,
                      "end": 4940
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4928,
                    "end": 4940
                  }
                }
              ],
              "loc": {
                "start": 4434,
                "end": 4942
              }
            },
            "loc": {
              "start": 4425,
              "end": 4942
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 4943,
                "end": 4945
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4943,
              "end": 4945
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4946,
                "end": 4956
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4946,
              "end": 4956
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4957,
                "end": 4967
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4957,
              "end": 4967
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 4968,
                "end": 4978
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4968,
              "end": 4978
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4979,
                "end": 4988
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4979,
              "end": 4988
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 4989,
                "end": 5000
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4989,
              "end": 5000
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 5001,
                "end": 5007
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
                      "start": 5017,
                      "end": 5027
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5014,
                    "end": 5027
                  }
                }
              ],
              "loc": {
                "start": 5008,
                "end": 5029
              }
            },
            "loc": {
              "start": 5001,
              "end": 5029
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 5030,
                "end": 5035
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
                      "value": "Organization",
                      "loc": {
                        "start": 5049,
                        "end": 5061
                      }
                    },
                    "loc": {
                      "start": 5049,
                      "end": 5061
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
                          "value": "Organization_nav",
                          "loc": {
                            "start": 5075,
                            "end": 5091
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5072,
                          "end": 5091
                        }
                      }
                    ],
                    "loc": {
                      "start": 5062,
                      "end": 5097
                    }
                  },
                  "loc": {
                    "start": 5042,
                    "end": 5097
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
                        "start": 5109,
                        "end": 5113
                      }
                    },
                    "loc": {
                      "start": 5109,
                      "end": 5113
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
                            "start": 5127,
                            "end": 5135
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5124,
                          "end": 5135
                        }
                      }
                    ],
                    "loc": {
                      "start": 5114,
                      "end": 5141
                    }
                  },
                  "loc": {
                    "start": 5102,
                    "end": 5141
                  }
                }
              ],
              "loc": {
                "start": 5036,
                "end": 5143
              }
            },
            "loc": {
              "start": 5030,
              "end": 5143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 5144,
                "end": 5155
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5144,
              "end": 5155
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 5156,
                "end": 5170
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5156,
              "end": 5170
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 5171,
                "end": 5176
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5171,
              "end": 5176
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 5177,
                "end": 5186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5177,
              "end": 5186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 5187,
                "end": 5191
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
                      "start": 5201,
                      "end": 5209
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5198,
                    "end": 5209
                  }
                }
              ],
              "loc": {
                "start": 5192,
                "end": 5211
              }
            },
            "loc": {
              "start": 5187,
              "end": 5211
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 5212,
                "end": 5226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5212,
              "end": 5226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 5227,
                "end": 5232
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5227,
              "end": 5232
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5233,
                "end": 5236
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
                      "start": 5243,
                      "end": 5253
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5243,
                    "end": 5253
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5258,
                      "end": 5267
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5258,
                    "end": 5267
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 5272,
                      "end": 5283
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5272,
                    "end": 5283
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5288,
                      "end": 5297
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5288,
                    "end": 5297
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5302,
                      "end": 5309
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5302,
                    "end": 5309
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 5314,
                      "end": 5322
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5314,
                    "end": 5322
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 5327,
                      "end": 5339
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5327,
                    "end": 5339
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 5344,
                      "end": 5352
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5344,
                    "end": 5352
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 5357,
                      "end": 5365
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5357,
                    "end": 5365
                  }
                }
              ],
              "loc": {
                "start": 5237,
                "end": 5367
              }
            },
            "loc": {
              "start": 5233,
              "end": 5367
            }
          }
        ],
        "loc": {
          "start": 4423,
          "end": 5369
        }
      },
      "loc": {
        "start": 4390,
        "end": 5369
      }
    },
    "Routine_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_nav",
        "loc": {
          "start": 5379,
          "end": 5390
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 5394,
            "end": 5401
          }
        },
        "loc": {
          "start": 5394,
          "end": 5401
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
                "start": 5404,
                "end": 5406
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5404,
              "end": 5406
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 5407,
                "end": 5417
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5407,
              "end": 5417
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5418,
                "end": 5427
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5418,
              "end": 5427
            }
          }
        ],
        "loc": {
          "start": 5402,
          "end": 5429
        }
      },
      "loc": {
        "start": 5370,
        "end": 5429
      }
    },
    "SmartContract_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "SmartContract_list",
        "loc": {
          "start": 5439,
          "end": 5457
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "SmartContract",
          "loc": {
            "start": 5461,
            "end": 5474
          }
        },
        "loc": {
          "start": 5461,
          "end": 5474
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
                "start": 5477,
                "end": 5485
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
                      "start": 5492,
                      "end": 5504
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
                            "start": 5515,
                            "end": 5517
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5515,
                          "end": 5517
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5526,
                            "end": 5534
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5526,
                          "end": 5534
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5543,
                            "end": 5554
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5543,
                          "end": 5554
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 5563,
                            "end": 5575
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5563,
                          "end": 5575
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5584,
                            "end": 5588
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5584,
                          "end": 5588
                        }
                      }
                    ],
                    "loc": {
                      "start": 5505,
                      "end": 5594
                    }
                  },
                  "loc": {
                    "start": 5492,
                    "end": 5594
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5599,
                      "end": 5601
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5599,
                    "end": 5601
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 5606,
                      "end": 5616
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5606,
                    "end": 5616
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 5621,
                      "end": 5631
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5621,
                    "end": 5631
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 5636,
                      "end": 5646
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5636,
                    "end": 5646
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 5651,
                      "end": 5660
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5651,
                    "end": 5660
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5665,
                      "end": 5673
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5665,
                    "end": 5673
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5678,
                      "end": 5687
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5678,
                    "end": 5687
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 5692,
                      "end": 5699
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5692,
                    "end": 5699
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contractType",
                    "loc": {
                      "start": 5704,
                      "end": 5716
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5704,
                    "end": 5716
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "content",
                    "loc": {
                      "start": 5721,
                      "end": 5728
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5721,
                    "end": 5728
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 5733,
                      "end": 5745
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5733,
                    "end": 5745
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 5750,
                      "end": 5762
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5750,
                    "end": 5762
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 5767,
                      "end": 5780
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5767,
                    "end": 5780
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 5785,
                      "end": 5807
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5785,
                    "end": 5807
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 5812,
                      "end": 5822
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5812,
                    "end": 5822
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 5827,
                      "end": 5839
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5827,
                    "end": 5839
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 5844,
                      "end": 5847
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
                            "start": 5858,
                            "end": 5868
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5858,
                          "end": 5868
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 5877,
                            "end": 5884
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5877,
                          "end": 5884
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 5893,
                            "end": 5902
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5893,
                          "end": 5902
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 5911,
                            "end": 5920
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5911,
                          "end": 5920
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 5929,
                            "end": 5938
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5929,
                          "end": 5938
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 5947,
                            "end": 5953
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5947,
                          "end": 5953
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 5962,
                            "end": 5969
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5962,
                          "end": 5969
                        }
                      }
                    ],
                    "loc": {
                      "start": 5848,
                      "end": 5975
                    }
                  },
                  "loc": {
                    "start": 5844,
                    "end": 5975
                  }
                }
              ],
              "loc": {
                "start": 5486,
                "end": 5977
              }
            },
            "loc": {
              "start": 5477,
              "end": 5977
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5978,
                "end": 5980
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5978,
              "end": 5980
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5981,
                "end": 5991
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5981,
              "end": 5991
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5992,
                "end": 6002
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5992,
              "end": 6002
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6003,
                "end": 6012
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6003,
              "end": 6012
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 6013,
                "end": 6024
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6013,
              "end": 6024
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 6025,
                "end": 6031
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
                      "start": 6041,
                      "end": 6051
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6038,
                    "end": 6051
                  }
                }
              ],
              "loc": {
                "start": 6032,
                "end": 6053
              }
            },
            "loc": {
              "start": 6025,
              "end": 6053
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 6054,
                "end": 6059
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
                      "value": "Organization",
                      "loc": {
                        "start": 6073,
                        "end": 6085
                      }
                    },
                    "loc": {
                      "start": 6073,
                      "end": 6085
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
                          "value": "Organization_nav",
                          "loc": {
                            "start": 6099,
                            "end": 6115
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6096,
                          "end": 6115
                        }
                      }
                    ],
                    "loc": {
                      "start": 6086,
                      "end": 6121
                    }
                  },
                  "loc": {
                    "start": 6066,
                    "end": 6121
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
                        "start": 6133,
                        "end": 6137
                      }
                    },
                    "loc": {
                      "start": 6133,
                      "end": 6137
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
                            "start": 6151,
                            "end": 6159
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6148,
                          "end": 6159
                        }
                      }
                    ],
                    "loc": {
                      "start": 6138,
                      "end": 6165
                    }
                  },
                  "loc": {
                    "start": 6126,
                    "end": 6165
                  }
                }
              ],
              "loc": {
                "start": 6060,
                "end": 6167
              }
            },
            "loc": {
              "start": 6054,
              "end": 6167
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 6168,
                "end": 6179
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6168,
              "end": 6179
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 6180,
                "end": 6194
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6180,
              "end": 6194
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 6195,
                "end": 6200
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6195,
              "end": 6200
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6201,
                "end": 6210
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6201,
              "end": 6210
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 6211,
                "end": 6215
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
                      "start": 6225,
                      "end": 6233
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6222,
                    "end": 6233
                  }
                }
              ],
              "loc": {
                "start": 6216,
                "end": 6235
              }
            },
            "loc": {
              "start": 6211,
              "end": 6235
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 6236,
                "end": 6250
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6236,
              "end": 6250
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 6251,
                "end": 6256
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6251,
              "end": 6256
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6257,
                "end": 6260
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
                      "start": 6267,
                      "end": 6276
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6267,
                    "end": 6276
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 6281,
                      "end": 6292
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6281,
                    "end": 6292
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 6297,
                      "end": 6308
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6297,
                    "end": 6308
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6313,
                      "end": 6322
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6313,
                    "end": 6322
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6327,
                      "end": 6334
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6327,
                    "end": 6334
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 6339,
                      "end": 6347
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6339,
                    "end": 6347
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6352,
                      "end": 6364
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6352,
                    "end": 6364
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 6369,
                      "end": 6377
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6369,
                    "end": 6377
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 6382,
                      "end": 6390
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6382,
                    "end": 6390
                  }
                }
              ],
              "loc": {
                "start": 6261,
                "end": 6392
              }
            },
            "loc": {
              "start": 6257,
              "end": 6392
            }
          }
        ],
        "loc": {
          "start": 5475,
          "end": 6394
        }
      },
      "loc": {
        "start": 5430,
        "end": 6394
      }
    },
    "SmartContract_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "SmartContract_nav",
        "loc": {
          "start": 6404,
          "end": 6421
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "SmartContract",
          "loc": {
            "start": 6425,
            "end": 6438
          }
        },
        "loc": {
          "start": 6425,
          "end": 6438
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
                "start": 6441,
                "end": 6443
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6441,
              "end": 6443
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6444,
                "end": 6453
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6444,
              "end": 6453
            }
          }
        ],
        "loc": {
          "start": 6439,
          "end": 6455
        }
      },
      "loc": {
        "start": 6395,
        "end": 6455
      }
    },
    "Standard_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_list",
        "loc": {
          "start": 6465,
          "end": 6478
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 6482,
            "end": 6490
          }
        },
        "loc": {
          "start": 6482,
          "end": 6490
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
                "start": 6493,
                "end": 6501
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
                      "start": 6508,
                      "end": 6520
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
                            "start": 6531,
                            "end": 6533
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6531,
                          "end": 6533
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6542,
                            "end": 6550
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6542,
                          "end": 6550
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6559,
                            "end": 6570
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6559,
                          "end": 6570
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 6579,
                            "end": 6591
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6579,
                          "end": 6591
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 6600,
                            "end": 6604
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6600,
                          "end": 6604
                        }
                      }
                    ],
                    "loc": {
                      "start": 6521,
                      "end": 6610
                    }
                  },
                  "loc": {
                    "start": 6508,
                    "end": 6610
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6615,
                      "end": 6617
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6615,
                    "end": 6617
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 6622,
                      "end": 6632
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6622,
                    "end": 6632
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 6637,
                      "end": 6647
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6637,
                    "end": 6647
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 6652,
                      "end": 6662
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6652,
                    "end": 6662
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isFile",
                    "loc": {
                      "start": 6667,
                      "end": 6673
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6667,
                    "end": 6673
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 6678,
                      "end": 6686
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6678,
                    "end": 6686
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6691,
                      "end": 6700
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6691,
                    "end": 6700
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 6705,
                      "end": 6712
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6705,
                    "end": 6712
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardType",
                    "loc": {
                      "start": 6717,
                      "end": 6729
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6717,
                    "end": 6729
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "props",
                    "loc": {
                      "start": 6734,
                      "end": 6739
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6734,
                    "end": 6739
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yup",
                    "loc": {
                      "start": 6744,
                      "end": 6747
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6744,
                    "end": 6747
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6752,
                      "end": 6764
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6752,
                    "end": 6764
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6769,
                      "end": 6781
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6769,
                    "end": 6781
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 6786,
                      "end": 6799
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6786,
                    "end": 6799
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 6804,
                      "end": 6826
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6804,
                    "end": 6826
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 6831,
                      "end": 6841
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6831,
                    "end": 6841
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 6846,
                      "end": 6858
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6846,
                    "end": 6858
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6863,
                      "end": 6866
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
                            "start": 6877,
                            "end": 6887
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6877,
                          "end": 6887
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 6896,
                            "end": 6903
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6896,
                          "end": 6903
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 6912,
                            "end": 6921
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6912,
                          "end": 6921
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 6930,
                            "end": 6939
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6930,
                          "end": 6939
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6948,
                            "end": 6957
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6948,
                          "end": 6957
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 6966,
                            "end": 6972
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6966,
                          "end": 6972
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 6981,
                            "end": 6988
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6981,
                          "end": 6988
                        }
                      }
                    ],
                    "loc": {
                      "start": 6867,
                      "end": 6994
                    }
                  },
                  "loc": {
                    "start": 6863,
                    "end": 6994
                  }
                }
              ],
              "loc": {
                "start": 6502,
                "end": 6996
              }
            },
            "loc": {
              "start": 6493,
              "end": 6996
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6997,
                "end": 6999
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6997,
              "end": 6999
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7000,
                "end": 7010
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7000,
              "end": 7010
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7011,
                "end": 7021
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7011,
              "end": 7021
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7022,
                "end": 7031
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7022,
              "end": 7031
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 7032,
                "end": 7043
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7032,
              "end": 7043
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 7044,
                "end": 7050
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
                      "start": 7060,
                      "end": 7070
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7057,
                    "end": 7070
                  }
                }
              ],
              "loc": {
                "start": 7051,
                "end": 7072
              }
            },
            "loc": {
              "start": 7044,
              "end": 7072
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 7073,
                "end": 7078
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
                      "value": "Organization",
                      "loc": {
                        "start": 7092,
                        "end": 7104
                      }
                    },
                    "loc": {
                      "start": 7092,
                      "end": 7104
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
                          "value": "Organization_nav",
                          "loc": {
                            "start": 7118,
                            "end": 7134
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7115,
                          "end": 7134
                        }
                      }
                    ],
                    "loc": {
                      "start": 7105,
                      "end": 7140
                    }
                  },
                  "loc": {
                    "start": 7085,
                    "end": 7140
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
                        "start": 7152,
                        "end": 7156
                      }
                    },
                    "loc": {
                      "start": 7152,
                      "end": 7156
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
                            "start": 7170,
                            "end": 7178
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7167,
                          "end": 7178
                        }
                      }
                    ],
                    "loc": {
                      "start": 7157,
                      "end": 7184
                    }
                  },
                  "loc": {
                    "start": 7145,
                    "end": 7184
                  }
                }
              ],
              "loc": {
                "start": 7079,
                "end": 7186
              }
            },
            "loc": {
              "start": 7073,
              "end": 7186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 7187,
                "end": 7198
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7187,
              "end": 7198
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 7199,
                "end": 7213
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7199,
              "end": 7213
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 7214,
                "end": 7219
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7214,
              "end": 7219
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7220,
                "end": 7229
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7220,
              "end": 7229
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 7230,
                "end": 7234
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
                      "start": 7244,
                      "end": 7252
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7241,
                    "end": 7252
                  }
                }
              ],
              "loc": {
                "start": 7235,
                "end": 7254
              }
            },
            "loc": {
              "start": 7230,
              "end": 7254
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 7255,
                "end": 7269
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7255,
              "end": 7269
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 7270,
                "end": 7275
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7270,
              "end": 7275
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7276,
                "end": 7279
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
                      "start": 7286,
                      "end": 7295
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7286,
                    "end": 7295
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 7300,
                      "end": 7311
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7300,
                    "end": 7311
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 7316,
                      "end": 7327
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7316,
                    "end": 7327
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7332,
                      "end": 7341
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7332,
                    "end": 7341
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7346,
                      "end": 7353
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7346,
                    "end": 7353
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 7358,
                      "end": 7366
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7358,
                    "end": 7366
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7371,
                      "end": 7383
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7371,
                    "end": 7383
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7388,
                      "end": 7396
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7388,
                    "end": 7396
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 7401,
                      "end": 7409
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7401,
                    "end": 7409
                  }
                }
              ],
              "loc": {
                "start": 7280,
                "end": 7411
              }
            },
            "loc": {
              "start": 7276,
              "end": 7411
            }
          }
        ],
        "loc": {
          "start": 6491,
          "end": 7413
        }
      },
      "loc": {
        "start": 6456,
        "end": 7413
      }
    },
    "Standard_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_nav",
        "loc": {
          "start": 7423,
          "end": 7435
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 7439,
            "end": 7447
          }
        },
        "loc": {
          "start": 7439,
          "end": 7447
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
                "start": 7450,
                "end": 7452
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7450,
              "end": 7452
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7453,
                "end": 7462
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7453,
              "end": 7462
            }
          }
        ],
        "loc": {
          "start": 7448,
          "end": 7464
        }
      },
      "loc": {
        "start": 7414,
        "end": 7464
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 7474,
          "end": 7482
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 7486,
            "end": 7489
          }
        },
        "loc": {
          "start": 7486,
          "end": 7489
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
                "start": 7492,
                "end": 7494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7492,
              "end": 7494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7495,
                "end": 7505
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7495,
              "end": 7505
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 7506,
                "end": 7509
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7506,
              "end": 7509
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7510,
                "end": 7519
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7510,
              "end": 7519
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 7520,
                "end": 7532
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
                      "start": 7539,
                      "end": 7541
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7539,
                    "end": 7541
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 7546,
                      "end": 7554
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7546,
                    "end": 7554
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 7559,
                      "end": 7570
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7559,
                    "end": 7570
                  }
                }
              ],
              "loc": {
                "start": 7533,
                "end": 7572
              }
            },
            "loc": {
              "start": 7520,
              "end": 7572
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7573,
                "end": 7576
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
                      "start": 7583,
                      "end": 7588
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7583,
                    "end": 7588
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7593,
                      "end": 7605
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7593,
                    "end": 7605
                  }
                }
              ],
              "loc": {
                "start": 7577,
                "end": 7607
              }
            },
            "loc": {
              "start": 7573,
              "end": 7607
            }
          }
        ],
        "loc": {
          "start": 7490,
          "end": 7609
        }
      },
      "loc": {
        "start": 7465,
        "end": 7609
      }
    },
    "User_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_list",
        "loc": {
          "start": 7619,
          "end": 7628
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7632,
            "end": 7636
          }
        },
        "loc": {
          "start": 7632,
          "end": 7636
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
                "start": 7639,
                "end": 7651
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
                      "start": 7658,
                      "end": 7660
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7658,
                    "end": 7660
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 7665,
                      "end": 7673
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7665,
                    "end": 7673
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 7678,
                      "end": 7681
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7678,
                    "end": 7681
                  }
                }
              ],
              "loc": {
                "start": 7652,
                "end": 7683
              }
            },
            "loc": {
              "start": 7639,
              "end": 7683
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7684,
                "end": 7686
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7684,
              "end": 7686
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7687,
                "end": 7697
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7687,
              "end": 7697
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7698,
                "end": 7708
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7698,
              "end": 7708
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7709,
                "end": 7720
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7709,
              "end": 7720
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7721,
                "end": 7727
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7721,
              "end": 7727
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7728,
                "end": 7733
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7728,
              "end": 7733
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7734,
                "end": 7738
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7734,
              "end": 7738
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7739,
                "end": 7751
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7739,
              "end": 7751
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7752,
                "end": 7761
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7752,
              "end": 7761
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsReceivedCount",
              "loc": {
                "start": 7762,
                "end": 7782
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7762,
              "end": 7782
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7783,
                "end": 7786
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
                      "start": 7793,
                      "end": 7802
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7793,
                    "end": 7802
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 7807,
                      "end": 7816
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7807,
                    "end": 7816
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7821,
                      "end": 7830
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7821,
                    "end": 7830
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7835,
                      "end": 7847
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7835,
                    "end": 7847
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7852,
                      "end": 7860
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7852,
                    "end": 7860
                  }
                }
              ],
              "loc": {
                "start": 7787,
                "end": 7862
              }
            },
            "loc": {
              "start": 7783,
              "end": 7862
            }
          }
        ],
        "loc": {
          "start": 7637,
          "end": 7864
        }
      },
      "loc": {
        "start": 7610,
        "end": 7864
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 7874,
          "end": 7882
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7886,
            "end": 7890
          }
        },
        "loc": {
          "start": 7886,
          "end": 7890
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
                "start": 7893,
                "end": 7895
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7893,
              "end": 7895
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7896,
                "end": 7906
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7896,
              "end": 7906
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7907,
                "end": 7917
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7907,
              "end": 7917
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7918,
                "end": 7929
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7918,
              "end": 7929
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7930,
                "end": 7936
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7930,
              "end": 7936
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7937,
                "end": 7942
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7937,
              "end": 7942
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7943,
                "end": 7947
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7943,
              "end": 7947
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7948,
                "end": 7960
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7948,
              "end": 7960
            }
          }
        ],
        "loc": {
          "start": 7891,
          "end": 7962
        }
      },
      "loc": {
        "start": 7865,
        "end": 7962
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
        "start": 7970,
        "end": 7978
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
              "start": 7980,
              "end": 7985
            }
          },
          "loc": {
            "start": 7979,
            "end": 7985
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
                "start": 7987,
                "end": 8005
              }
            },
            "loc": {
              "start": 7987,
              "end": 8005
            }
          },
          "loc": {
            "start": 7987,
            "end": 8006
          }
        },
        "directives": [],
        "loc": {
          "start": 7979,
          "end": 8006
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
              "start": 8012,
              "end": 8020
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 8021,
                  "end": 8026
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 8029,
                    "end": 8034
                  }
                },
                "loc": {
                  "start": 8028,
                  "end": 8034
                }
              },
              "loc": {
                "start": 8021,
                "end": 8034
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
                    "start": 8042,
                    "end": 8047
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
                          "start": 8058,
                          "end": 8064
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8058,
                        "end": 8064
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 8073,
                          "end": 8077
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
                                  "start": 8099,
                                  "end": 8102
                                }
                              },
                              "loc": {
                                "start": 8099,
                                "end": 8102
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
                                      "start": 8124,
                                      "end": 8132
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8121,
                                    "end": 8132
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8103,
                                "end": 8146
                              }
                            },
                            "loc": {
                              "start": 8092,
                              "end": 8146
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
                                  "start": 8166,
                                  "end": 8170
                                }
                              },
                              "loc": {
                                "start": 8166,
                                "end": 8170
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
                                      "start": 8192,
                                      "end": 8201
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8189,
                                    "end": 8201
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8171,
                                "end": 8215
                              }
                            },
                            "loc": {
                              "start": 8159,
                              "end": 8215
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Organization",
                                "loc": {
                                  "start": 8235,
                                  "end": 8247
                                }
                              },
                              "loc": {
                                "start": 8235,
                                "end": 8247
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
                                    "value": "Organization_list",
                                    "loc": {
                                      "start": 8269,
                                      "end": 8286
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8266,
                                    "end": 8286
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8248,
                                "end": 8300
                              }
                            },
                            "loc": {
                              "start": 8228,
                              "end": 8300
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
                                  "start": 8320,
                                  "end": 8327
                                }
                              },
                              "loc": {
                                "start": 8320,
                                "end": 8327
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
                                      "start": 8349,
                                      "end": 8361
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8346,
                                    "end": 8361
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8328,
                                "end": 8375
                              }
                            },
                            "loc": {
                              "start": 8313,
                              "end": 8375
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
                                  "start": 8395,
                                  "end": 8403
                                }
                              },
                              "loc": {
                                "start": 8395,
                                "end": 8403
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
                                      "start": 8425,
                                      "end": 8438
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8422,
                                    "end": 8438
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8404,
                                "end": 8452
                              }
                            },
                            "loc": {
                              "start": 8388,
                              "end": 8452
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
                                  "start": 8472,
                                  "end": 8479
                                }
                              },
                              "loc": {
                                "start": 8472,
                                "end": 8479
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
                                      "start": 8501,
                                      "end": 8513
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8498,
                                    "end": 8513
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8480,
                                "end": 8527
                              }
                            },
                            "loc": {
                              "start": 8465,
                              "end": 8527
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "SmartContract",
                                "loc": {
                                  "start": 8547,
                                  "end": 8560
                                }
                              },
                              "loc": {
                                "start": 8547,
                                "end": 8560
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
                                    "value": "SmartContract_list",
                                    "loc": {
                                      "start": 8582,
                                      "end": 8600
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8579,
                                    "end": 8600
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8561,
                                "end": 8614
                              }
                            },
                            "loc": {
                              "start": 8540,
                              "end": 8614
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
                                  "start": 8634,
                                  "end": 8642
                                }
                              },
                              "loc": {
                                "start": 8634,
                                "end": 8642
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
                                      "start": 8664,
                                      "end": 8677
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8661,
                                    "end": 8677
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8643,
                                "end": 8691
                              }
                            },
                            "loc": {
                              "start": 8627,
                              "end": 8691
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
                                  "start": 8711,
                                  "end": 8715
                                }
                              },
                              "loc": {
                                "start": 8711,
                                "end": 8715
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
                                      "start": 8737,
                                      "end": 8746
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8734,
                                    "end": 8746
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8716,
                                "end": 8760
                              }
                            },
                            "loc": {
                              "start": 8704,
                              "end": 8760
                            }
                          }
                        ],
                        "loc": {
                          "start": 8078,
                          "end": 8770
                        }
                      },
                      "loc": {
                        "start": 8073,
                        "end": 8770
                      }
                    }
                  ],
                  "loc": {
                    "start": 8048,
                    "end": 8776
                  }
                },
                "loc": {
                  "start": 8042,
                  "end": 8776
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 8781,
                    "end": 8789
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
                          "start": 8800,
                          "end": 8811
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8800,
                        "end": 8811
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorApi",
                        "loc": {
                          "start": 8820,
                          "end": 8832
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8820,
                        "end": 8832
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorNote",
                        "loc": {
                          "start": 8841,
                          "end": 8854
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8841,
                        "end": 8854
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorOrganization",
                        "loc": {
                          "start": 8863,
                          "end": 8884
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8863,
                        "end": 8884
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 8893,
                          "end": 8909
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8893,
                        "end": 8909
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorQuestion",
                        "loc": {
                          "start": 8918,
                          "end": 8935
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8918,
                        "end": 8935
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRoutine",
                        "loc": {
                          "start": 8944,
                          "end": 8960
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8944,
                        "end": 8960
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorSmartContract",
                        "loc": {
                          "start": 8969,
                          "end": 8991
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8969,
                        "end": 8991
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorStandard",
                        "loc": {
                          "start": 9000,
                          "end": 9017
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9000,
                        "end": 9017
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorUser",
                        "loc": {
                          "start": 9026,
                          "end": 9039
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9026,
                        "end": 9039
                      }
                    }
                  ],
                  "loc": {
                    "start": 8790,
                    "end": 9045
                  }
                },
                "loc": {
                  "start": 8781,
                  "end": 9045
                }
              }
            ],
            "loc": {
              "start": 8036,
              "end": 9049
            }
          },
          "loc": {
            "start": 8012,
            "end": 9049
          }
        }
      ],
      "loc": {
        "start": 8008,
        "end": 9051
      }
    },
    "loc": {
      "start": 7964,
      "end": 9051
    }
  },
  "variableValues": {},
  "path": {
    "key": "popular_findMany"
  }
} as const;
